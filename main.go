package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	seenInc := map[string]bool{}
	protocArgs := []string{"--proto_path=."}
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
		pkg := struct {
			Target string
			Root   string
		}{}
		if err := goListPkg(importPath, &pkg); err != nil {
			continue
		}
		if pkg.Root == "" || !filepath.IsAbs(pkg.Target) {
			continue
		}
		if err := goInstall(importPath); err != nil {
			stderr.Println(pkg, err)
			continue
		}
		if argVal != "" {
			inputArgs[argPos] = argNm + "=" + insertGogoTypes(argVal)
		}
		if !seenInc[pkg.Root] {
			seenInc[pkg.Root] = true
			protocArgs = append(protocArgs, "--proto_path="+filepath.Join(pkg.Root, "protobuf"))
			protocArgs = append(protocArgs, "--proto_path="+pkg.Root)
		}
		protocArgs = append(protocArgs, "--plugin="+name+"="+pkg.Target)
	}
	protocArgs = append(protocArgs, inputArgs...)

	protocExe, err := exec.LookPath("protoc")
	if err != nil {
		stderr.Fatalln("Cannot find the `protoc` command.\nPlease install it or see: https://github.com/protocolbuffers/protobuf#protocol-compiler-installation")
	}

	cmd := exec.Command(protocExe, protocArgs...)
	cmd.Stdout = os.Stdout
	if err := runCmd(cmd); err != nil {
		stderr.Fatalln(err)
	}
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
