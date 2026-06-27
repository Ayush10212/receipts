package sink

import (
	"context"
	"encoding/json"
	"io"

	"github.com/Ayush10212/receipts/core/report"
)

// SARIF 2.1.0 minimal types for receipts output.
type sarifLog struct {
	Version string      `json:"version"`
	Schema  string      `json:"$schema"`
	Runs    []sarifRun  `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	InformationURI string `json:"informationUri"`
}

type sarifResult struct {
	RuleID  string          `json:"ruleId"`
	Level   string          `json:"level"`
	Message sarifMessage    `json:"message"`
	Locs    []sarifLocation `json:"locations,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
	EndLine     int `json:"endLine,omitempty"`
	EndColumn   int `json:"endColumn,omitempty"`
}

// SARIFSink emits a SARIF 2.1.0 report to w.
type SARIFSink struct{ W io.Writer }

func (s SARIFSink) Emit(_ context.Context, _ report.ExecutionContext, r report.Report) error {
	var results []sarifResult
	for _, c := range r.Claims {
		if c.Verdict == report.VerdictGrounded {
			continue
		}
		level := "warning"
		if c.Verdict == report.VerdictContradicted {
			level = "error"
		}

		detail := c.Text
		for _, e := range c.Evidence {
			if e.Detail != "" {
				detail = e.Detail
				break
			}
		}

		res := sarifResult{
			RuleID:  string(c.Verdict),
			Level:   level,
			Message: sarifMessage{Text: detail},
			Locs: []sarifLocation{
				{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{URI: c.Locus.File},
						Region: sarifRegion{
							StartLine:   c.Locus.Line,
							StartColumn: c.Locus.Col,
							EndLine:     c.Locus.EndLine,
							EndColumn:   c.Locus.EndCol,
						},
					},
				},
			},
		}
		results = append(results, res)
	}

	log := sarifLog{
		Version: "2.1.0",
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:           "receipts",
						Version:        r.Run.ToolVersion,
						InformationURI: "https://github.com/Ayush10212/receipts",
					},
				},
				Results: results,
			},
		},
	}

	enc := json.NewEncoder(s.W)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}
