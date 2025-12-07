package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/woozymasta/dzmap/internal/geo"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v3"
)

type Options struct {
	Input  string  `short:"i" long:"in" description:"Input file path (cfgNames.hpp). Reads from stdin if empty"`
	Output string  `short:"o" long:"out" description:"Output file path. Writes to stdout if empty"`
	Format string  `short:"f" long:"format" description:"Output format" choice:"json" choice:"yaml" default:"json"`
	Size   float64 `short:"s" long:"size" description:"Map size in meters (e.g. 15360 for Chernarus)" required:"true"`
}

// Regex Pattern captures: 1=Name, 2=X, 3=Z, 4=Type
var cfgRegex = regexp.MustCompile(
	`class\s+\w+\s*\{` + // Start of class block (e.g. "class City {")
		`[\s\S]*?` + // Non-greedy skip (matches across newlines)
		`name\s*=\s*"([^"]+)";` + // Group 1: Name
		`[\s\S]*?` + // Skip content
		`position\[\]\s*=\s*\{` + // Start of position array
		`([\d\.]+),\s*([\d\.]+)` + // Group 2 & 3: X and Z coordinates
		`\};` + // End of position array
		`[\s\S]*?` + // Skip content
		`type\s*=\s*"([^"]+)";` + // Group 4: Type
		`[\s\S]*?\};`, // End of class block
)

func main() {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if opts.Size <= 0 {
		fmt.Fprintln(os.Stderr, "Error: --size must be > 0")
		os.Exit(1)
	}

	// Read Input
	var inputData []byte
	var err error

	if opts.Input != "" {
		inputData, err = os.ReadFile(opts.Input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
			os.Exit(1)
		}
	} else {
		inputData, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
	}

	content := string(inputData)
	matches := cfgRegex.FindAllStringSubmatch(content, -1)

	fc := geo.GeoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Features: make([]geo.GeoJSONFeature, 0, len(matches)),
	}

	count := 0
	for _, match := range matches {
		name := match[1]
		xStr := match[2]
		zStr := match[3]
		typeStr := match[4]

		x, err1 := strconv.ParseFloat(xStr, 64)
		z, err2 := strconv.ParseFloat(zStr, 64)

		if err1 != nil || err2 != nil {
			fmt.Fprintf(os.Stderr, "Skipping %s due to invalid coords: %s, %s\n", name, xStr, zStr)
			continue
		}

		lon, lat := geo.GameToMetricZ(x, z, opts.Size)

		feature := geo.GeoJSONFeature{
			Type: "Feature",
			Geometry: geo.GeoJSONGeometry{
				Type:        "Point",
				Coordinates: []float64{lon, lat},
			},
			Properties: map[string]interface{}{
				"name": name,
				"type": strings.ToLower(typeStr),
			},
		}

		fc.Features = append(fc.Features, feature)
		count++
	}

	// marshal
	var outputData []byte
	if opts.Format == "yaml" {
		outputData, err = yaml.Marshal(fc)
	} else {
		outputData, err = json.MarshalIndent(fc, "", "  ")
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling data: %v\n", err)
		os.Exit(1)
	}

	if opts.Output != "" {
		err = os.WriteFile(opts.Output, outputData, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Successfully converted %d locations to %s (format: %s)\n", count, opts.Output, opts.Format)
	} else {
		fmt.Println(string(outputData))
	}
}
