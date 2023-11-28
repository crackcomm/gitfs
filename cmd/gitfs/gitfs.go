// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	gitfs "github.com/gravypod/gitfs/pkg"
	"github.com/jacobsa/fuse"
)

var (
	branch              = flag.String("branch", "", "Branch to mount.")
	repositoryDirectory = flag.String("git-dir", "", "Path to bare git repo to serve.")
	mountPath           = flag.String("mount", "/tmp/gitfs", "Location to mount gitfs. You must have write access to this directory.")
)

func main() {
	flag.Parse()

	if *repositoryDirectory == "" {
		log.Fatalf("Must provide a bare git repository (--git-dir)")
	}

	if *mountPath == "" {
		log.Fatalf("Must provide a location to mount into (--mount)")
	}

	if _, err := os.Stat(*mountPath); os.IsNotExist(err) {
		err := os.Mkdir(*mountPath, os.FileMode(0444))
		if err != nil {
			log.Fatalf("Could not create mount path: %v", err)
		}
	}

	absoluteMountPath, err := filepath.Abs(*mountPath)
	if err != nil {
		log.Fatalf("failed to resolve path: %v", err)
	}
	log.Printf("Attempting to mount to %s", absoluteMountPath)

	config := fuse.MountConfig{
		ReadOnly:                  true,
		DisableWritebackCaching:   true,
		EnableSymlinkCaching:      false,
		DisableDefaultPermissions: true,

		DebugLogger: log.New(os.Stderr, "fuse debug: ", 0),
		ErrorLogger: log.New(os.Stderr, "fuse error: ", 0),
	}

	git, err := gitfs.NewCliGit(*repositoryDirectory)
	if err != nil {
		log.Fatalf("Failed to create git client for directory '%s': %v", *repositoryDirectory,
			err)
	}

	fs := gitfs.NewReferenceFileSystem(git, gitfs.GitReference{Branch: branch})

	server, err := gitfs.NewBillyFuseServer(fs)
	if err != nil {
		log.Fatalf("Failed to start go-billy server: %v", err)
	}
	log.Println("Server started")

	mounted, err := fuse.Mount(absoluteMountPath, server, &config)
	if err != nil {
		log.Fatalf("Mount failed: %v", err)
	}

	err = mounted.Join(context.Background())
	if err != nil {
		log.Fatalf("Mount crashed: %v", err)
	}

}
