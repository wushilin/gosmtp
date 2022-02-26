package main

import (
	"log"
	"os"
)

func CreateDirectoryIfNotThere(dir string) {
	stat, err := os.Stat(dir)
	if err != nil {
		log.Printf("Folder %s doesn't exist. Creating..", dir)
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Fatalf("Can't create folder %s, error: %s", dir, err)
		}
	} else {
		if !stat.IsDir() {
			log.Fatalf("%s is not a directory!", dir)
		}
	}

}
