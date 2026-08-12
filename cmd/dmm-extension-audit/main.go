package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

type auditRow struct {
	Extension string `json:"extension"`
	Game      string `json:"game,omitempty"`
	Surface   string `json:"surface"`
	ID        string `json:"id"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Coverage  string `json:"coverage,omitempty"`
}

func main() {
	format := flag.String("format", "text", "output format: text or json")
	includeReady := flag.Bool("ready", false, "include ready capabilities")
	flag.Parse()

	registry := gameext.NewRegistry(extensions.FirstParty())
	rows := auditSummaries(registry.ExtensionSummaries(), *includeReady)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Extension != rows[j].Extension {
			return rows[i].Extension < rows[j].Extension
		}
		if rows[i].Surface != rows[j].Surface {
			return rows[i].Surface < rows[j].Surface
		}
		return rows[i].ID < rows[j].ID
	})

	switch *format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(rows); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "text":
		for _, row := range rows {
			message := strings.TrimSpace(row.Message)
			if message != "" {
				message = " - " + message
			}
			fmt.Printf("%s\t%s\t%s\t%s%s\n", row.Extension, row.Surface, row.ID, row.Status, message)
		}
		fmt.Fprintf(os.Stderr, "rows=%d\n", len(rows))
	default:
		fmt.Fprintln(os.Stderr, "format must be text or json")
		os.Exit(2)
	}
}

func auditSummaries(summaries []gameext.ExtensionSummary, includeReady bool) []auditRow {
	var rows []auditRow
	for _, summary := range summaries {
		if includeReady || summary.Coverage == gameext.CoverageResearchBlocked || summary.Coverage == gameext.CoverageMetadataOnly {
			rows = append(rows, auditRow{
				Extension: summary.ID,
				Game:      summary.Name,
				Surface:   "extension",
				ID:        summary.ID,
				Status:    summary.Coverage,
				Coverage:  summary.Coverage,
			})
		}
		for _, gap := range summary.ParityGaps {
			rows = append(rows, auditRow{
				Extension: summary.ID,
				Game:      summary.Name,
				Surface:   gap.Surface,
				ID:        gap.ID,
				Status:    gap.Status,
				Message:   gap.Message,
				Coverage:  summary.Coverage,
			})
		}
		rows = append(rows, featureRows(summary.ID, summary.Name, summary.Coverage, summary.Capabilities, includeReady)...)
	}
	return rows
}

func featureRows(extensionID, gameName, coverage string, capabilities gameext.ExtensionCapabilities, includeReady bool) []auditRow {
	value := reflect.ValueOf(capabilities)
	typ := value.Type()
	var rows []auditRow
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() != reflect.Slice || field.Type().Elem() != reflect.TypeOf(gameext.FeatureSummary{}) {
			continue
		}
		surface := jsonSurfaceName(typ.Field(i))
		for idx := 0; idx < field.Len(); idx++ {
			feature := field.Index(idx).Interface().(gameext.FeatureSummary)
			rows = append(rows, auditFeature(extensionID, gameName, coverage, surface, feature, includeReady)...)
		}
	}
	return rows
}

func auditFeature(extensionID, gameName, coverage, surface string, feature gameext.FeatureSummary, includeReady bool) []auditRow {
	status := strings.TrimSpace(feature.Status)
	if status == "" {
		status = sdk.CapabilityStatusReady
	}
	row := auditRow{
		Extension: extensionID,
		Game:      gameName,
		Surface:   surface,
		ID:        feature.ID,
		Status:    status,
		Message:   feature.Message,
		Coverage:  coverage,
	}
	var rows []auditRow
	if includeReady || status != sdk.CapabilityStatusReady {
		rows = append(rows, row)
	}
	for _, child := range feature.Variants {
		childRows := auditFeature(extensionID, gameName, coverage, surface+"/variants", child, includeReady)
		rows = append(rows, childRows...)
	}
	for _, child := range feature.Commands {
		childRows := auditFeature(extensionID, gameName, coverage, surface+"/commands", child, includeReady)
		rows = append(rows, childRows...)
	}
	return rows
}

func jsonSurfaceName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	name, _, _ := strings.Cut(tag, ",")
	if name != "" && name != "-" {
		return name
	}
	return strings.ToLower(field.Name)
}
