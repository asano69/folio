package main

import (
	"fmt"
	"os"

	"folio/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cfg := config.Load()

	switch os.Args[1] {
	case "server":
		runServer(cfg)
	case "scan":
		scanPath := cfg.LibraryPath
		if len(os.Args) >= 3 {
			scanPath = os.Args[2]
		}
		if err := runScan(cfg, scanPath); err != nil {
			fmt.Fprintf(os.Stderr, "scan: %v\n", err)
			os.Exit(1)
		}
	case "thumbnail":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: folio thumbnail <book-uuid>\n")
			os.Exit(1)
		}
		if err := runThumbnail(cfg, os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "thumbnail: %v\n", err)
			os.Exit(1)
		}
	case "page-thumbnails":
		bookID := ""
		if len(os.Args) >= 3 {
			bookID = os.Args[2]
		}
		if err := runImageThumbnails(cfg, bookID); err != nil {
			fmt.Fprintf(os.Stderr, "page-thumbnails: %v\n", err)
			os.Exit(1)
		}
	case "hash":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: folio hash <book-uuid>\n")
			os.Exit(1)
		}
		if err := runHash(cfg, os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "hash: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: folio [server|scan [path]|thumbnail <uuid>|page-thumbnails [uuid]|hash <uuid>]\n")
}
