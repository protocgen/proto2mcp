// Package main implements the protoc-gen-proto2mcp plugin.
package main

import (
	"log"

	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	protogen.Options{}.Run(func(gen *protogen.Plugin) error {
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}

			for _, service := range f.Services {
				log.Printf("TODO: Process service %s", service.GoName)
				for _, method := range service.Methods {
					log.Printf("TODO: Process method %s", method.GoName)
				}
			}
		}
		return nil
	})
}
