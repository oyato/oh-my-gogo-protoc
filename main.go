package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	stderr    = log.New(os.Stderr, "omg-protoc: ", 0)
	gogoTypes = strings.Join([]string{
		"Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types",
		"Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types",
		"Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types",
		"Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types",
		"Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types",
		"Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types",
	}, ",")
)

func main() {
	if err := Main(); err != nil {
		stderr.Fatalln(err)
	}
}

func Main() error {
	gg := struct{ Root string }{}
	err := goListPkg("github.com/gogo/protobuf/gogoproto", &gg)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(gg.Root) {
		return fmt.Errorf("Cannot find github.com/gogo/protobuf module root")
	}

	tempDir, err := ioutil.TempDir("", "omg-protoc-")
	if err != nil {
		return fmt.Errorf("Cannot temp dir: %s", err)
	}
	defer os.RemoveAll(tempDir)

	protoDstDir := filepath.Join(tempDir, "github.com", "gogo", "protobuf", "gogoproto")
	protoSrcDir := filepath.Join(gg.Root, "gogoproto")
	if err := copyProtoFiles(protoDstDir, protoSrcDir); err != nil {
		return fmt.Errorf("Cannot copy gogoproto/*.proto files: %s", err)
	}

	protocArgs := []string{
		"--proto_path=.",
		"--proto_path=" + tempDir,
		"--proto_path=" + filepath.Join(gg.Root, "protobuf"),
	}

	inputArgs := append([]string{}, os.Args[1:]...)
	for argPos, arg := range inputArgs {
		argL := strings.SplitN(arg, "=", 2)
		argNm := argL[0]
		argVal := ""
		if len(argL) == 2 {
			argVal = argL[1]
		}
		if !strings.HasPrefix(argNm, "--") || !strings.HasSuffix(argNm, "_out") {
			continue
		}

		if argNm == "--twirp_out" {
			goInstall("github.com/twitchtv/twirp/protoc-gen-twirp")
			continue
		}

		name := "protoc-gen-" + argNm[2:len(argNm)-4]
		importPath := "github.com/gogo/protobuf/" + name
		pkg := struct{ Target string }{}
		if err := goListPkg(importPath, &pkg); err != nil {
			continue
		}
		if !filepath.IsAbs(pkg.Target) {
			continue
		}
		if err := goInstall(importPath); err != nil {
			stderr.Println(pkg, err)
			continue
		}
		if argVal != "" {
			inputArgs[argPos] = argNm + "=" + insertGogoTypes(argVal)
		}
		protocArgs = append(protocArgs, "--plugin="+name+"="+pkg.Target)
	}
	protocArgs = append(protocArgs, inputArgs...)

	protocExe, err := exec.LookPath("protoc")
	if err != nil {
		return fmt.Errorf("Cannot find the `protoc` command: %s.\nPlease install it or see: https://github.com/protocolbuffers/protobuf#protocol-compiler-installation", err)
	}

	cmd := exec.Command(protocExe, protocArgs...)
	cmd.Stdout = os.Stdout
	if err := runCmd(cmd); err != nil {
		return err
	}
	return nil
}

func copyProtoFiles(dstDir string, srcDir string) error {
	os.MkdirAll(dstDir, 0755)

	srcFiles, err := filepath.Glob(filepath.Join(srcDir, "*.proto"))
	if err != nil {
		return fmt.Errorf("Cannot glob .proto files in %s: %s", srcDir, err)
	}
	if len(srcFiles) == 0 {
		return fmt.Errorf("Cannot find any .proto files in %s", srcDir)
	}

	for _, srcFn := range srcFiles {
		src, err := ioutil.ReadFile(srcFn)
		if err != nil {
			return fmt.Errorf("Cannot read %s: %s", srcFn, err)
		}
		dstFn := filepath.Join(dstDir, filepath.Base(srcFn))
		err = ioutil.WriteFile(dstFn, src, 0644)
		if err != nil {
			return fmt.Errorf("Cannot write to %s: %s", dstFn, err)
		}
	}
	return nil
}

func insertGogoTypes(argVal string) string {
	i := strings.LastIndexByte(argVal, ':')
	if i < 0 {
		return gogoTypes + ":" + argVal
	}
	return gogoTypes + "," + strings.TrimLeft(argVal, ", ")
}

func goInstall(importPath string) error {
	cmd := exec.Command("go", "install", "-mod=readonly", importPath)
	return runCmd(cmd)
}

func goListPkg(importPath string, res interface{}) error {
	cmd := exec.Command("go", "list", "-mod=readonly", "-json", importPath)
	outBuf := &bytes.Buffer{}
	cmd.Stdout = outBuf
	if err := runCmd(cmd); err != nil {
		return err
	}
	if err := json.Unmarshal(outBuf.Bytes(), res); err != nil {
		return fmt.Errorf("Cannot parse `go list` output: %s", err)
	}
	return nil
}

func runCmd(cmd *exec.Cmd) error {
	errBuf := &bytes.Buffer{}
	cmd.Stderr = errBuf
	err := cmd.Run()
	if err == nil {
		return nil
	}
	return fmt.Errorf("`%s` failed: %s\n  %s",
		cmd, err,
		bytes.ReplaceAll(bytes.TrimSpace(errBuf.Bytes()), []byte("\n"), []byte("\n  ")),
	)
}
