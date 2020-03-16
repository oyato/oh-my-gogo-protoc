# oh-my-gogo-protoc

## What it is?

oh-my-gogo-protoc is a wrapper around the protobuf compiler (`protoc`) that automatically sets it up for use with https://github.com/gogo/protobuf and Go modules.

## How does it work?

- Given your _current_ command `protoc ... --gofast_out=Mgoogle...:. my.proto`, it sees you're using the `gofast` generator and:
- Automatically runs `go install github.com/gogo/protobuf/protoc-gen-gofast`.
- Then it runs `protoc --proto_path=...github.com/gogo/protobuf/protobuf@1... my.proto` i.e. with the correct include paths\*\* and the `google/protobuf/*.proto=github.com/gogo/protobuf/types` mappings.

Replace `gofast_out` with `gogoslick_out` or any other generator you prefer. It will figure it out and install and set up the appropriate binary.

## Why should I use it?

It makes working with Go modules and gogo/protobuf easy and pain-free.

Typically, to use gogo/protobuf you run protoc with a bunch of params like `-I$GOPATH/src/...`. This doesn't work for Go modules because the directory is at `$GOPATH/pkg/mod/github.com/gogo/protobuf@version/...` so you need a work-around involving `go list -f {{.Dir}}...` which may or may not work. It also doesn't work if `$GOPATH` is a list of directories.

## How do I install and use it?

- In your module dir, run `go get oya.to/oh-my-gogo-protoc`.
- In your source file add or edit your [go generate](https://golang.org/cmd/go/#hdr-Generate_Go_files_by_processing_source) directive `//go:generate go run oya.to/oh-my-gogo-protoc --gofast_out=. my.proto`. That's it.
- _Don't forget to install the gogo/proto deps you need e.g. `go get github.com/gogo/protobuf/protoc-gen-gofast`_

## Can I use it with [Twirp](https://github.com/twitchtv/twirp) RPC?

Yes, if you pass `--twirp_out`, it knows to automatically `go install github.com/twitchtv/twirp/protoc-gen-twirp` as well.

## Is it production ready?

Yes! we've literally been using it to generate over 1 go package since an hour ago, so you know it's battle tested.
