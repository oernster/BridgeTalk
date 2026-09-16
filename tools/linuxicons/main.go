// Command linuxicons installs the application's icons where a Linux desktop looks for them: every
// picture in the committed .ico, each under the hicolor theme at its own size and named for the
// application id (FR-810). The .ico is the one icon file, so no size list or second copy of the
// artwork lives anywhere else.
//
// build_flatpak.sh runs it inside the sandbox from the repository root:
//
//	go run ./tools/linuxicons -prefix /app
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/oernster/bridge-talk/internal/infrastructure/iconfile"
	"github.com/oernster/bridge-talk/internal/product"
)

const (
	// iconPath is the committed icon, relative to the repository root.
	iconPath = "assets/application-icon.ico"
	// filePerm makes an icon readable by everyone, writable by the owner.
	filePerm = 0o644
	// dirPerm makes an icon directory readable and searchable by everyone, writable by the owner.
	dirPerm = 0o755
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "linuxicons:", err)
		os.Exit(1)
	}
}

// run reads the flags, then installs every picture in the icon under the prefix, naming each one it
// wrote.
func run(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("linuxicons", flag.ContinueOnError)
	flags.SetOutput(out)
	prefix := flags.String("prefix", "", "the install prefix the icons go under, such as /app")
	icon := flags.String("icon", iconPath, "the .ico to read")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *prefix == "" {
		return fmt.Errorf("-prefix is required")
	}
	ico, err := os.ReadFile(*icon)
	if err != nil {
		return fmt.Errorf("reading %s: %w", *icon, err)
	}
	written, err := install(ico, *prefix)
	for _, path := range written {
		fmt.Fprintln(out, path)
	}
	return err
}

// install writes every square picture in an .ico under prefix/share/icons/hicolor, answering the
// paths it wrote.
func install(ico []byte, prefix string) ([]string, error) {
	sides, err := iconfile.Sides(ico)
	if err != nil {
		return nil, err
	}
	if len(sides) == 0 {
		return nil, fmt.Errorf("the icon holds no square picture")
	}
	var written []string
	for _, side := range sides {
		frame, err := iconfile.Frame(ico, side)
		if err != nil {
			return written, err
		}
		size := strconv.Itoa(side) + "x" + strconv.Itoa(side)
		path := filepath.Join(prefix, "share", "icons", "hicolor", size, "apps", product.AppID+".png")
		if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
			return written, fmt.Errorf("making %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, frame, filePerm); err != nil {
			return written, fmt.Errorf("writing %s: %w", path, err)
		}
		written = append(written, path)
	}
	return written, nil
}
