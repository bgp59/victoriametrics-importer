// Random parser tests.
// Simply display values for sanity check:

package parser

import (
	"bytes"
	"fmt"

	"testing"
)

func newTestableRandomParser(parserCfg any) (parser any, desc string, err error) {
	switch cfg := parserCfg.(type) {
	case *RandomGaugeParserConfig:
		parser = NewRandomGaugeParser(cfg)
		desc = fmt.Sprintf("Gauge{Range: %d .. %d, Rpt: 1 .. %d, Seed: %d}", cfg.Min, cfg.Max, cfg.MaxRepeat, cfg.Seed)
	case *RandomCounterParserConfig:
		parser = NewRandomCounterParser(cfg)
		desc = fmt.Sprintf("Counter{Init: %d, Inc: +%d .. %d, Rpt: 1 .. %d, Seed: %d}", cfg.Init, cfg.MinInc, cfg.MaxInc, cfg.MaxRepeat, cfg.Seed)
	case *RandomCategoricalParserConfig:
		parser = NewRandomCategoricalParser(cfg)
		desc = fmt.Sprintf("Categorical{#cat: %d, Rpt: 1 .. %d, Seed: %d}", len(cfg.Choices), cfg.MaxRepeat, cfg.Seed)
	default:
		err = fmt.Errorf("invalid cfg type %T", parserCfg)
	}
	return
}

func testRandomParser(t *testing.T, numParse int, cfgs []any) {
	parsers := make([]any, len(cfgs))
	descriptions := make([]string, len(cfgs))
	values := make([][]string, len(cfgs))
	descWidth := 0
	for i, cfg := range cfgs {
		parser, desc, err := newTestableRandomParser(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if len(desc) > descWidth {
			descWidth = len(desc)
		}
		parsers[i] = parser
		descriptions[i] = desc
		values[i] = make([]string, 0)
	}
	for i, desc := range descriptions {
		descriptions[i] = fmt.Sprintf("%*s", descWidth, desc)
	}

	value, valueWidth := "", 0
	for range numParse {
		for i, parser := range parsers {
			switch p := parser.(type) {
			case *RandomGaugeParser:
				p.Parse()
				value = string(p.ValBytes)
			case *RandomCounterParser:
				p.Parse()
				value = fmt.Sprintf("%d", p.Val)
			case *RandomCategoricalParser:
				p.Parse()
				value = string(p.Val)
			default:
				t.Fatalf("invalid parser type: %T", parser)
			}
			values[i] = append(values[i], value)
			if len(value) > valueWidth {
				valueWidth = len(value)
			}
		}
	}
	valueWidth += 1

	numValsPerLine := max(int(70/valueWidth), 1)
	buf := &bytes.Buffer{}
	for k := 0; k < numParse; k += numValsPerLine {
		lines := make([]string, len(parsers))
		for i := range parsers {
			lines[i] = "\n\t" + descriptions[i] + ":"
			parserValues := values[i]
			for j := range min(numParse-k, numValsPerLine) {
				lines[i] += fmt.Sprintf("%*s", valueWidth, parserValues[k+j])
			}
		}

		for _, line := range lines {
			buf.WriteString(line)
		}
		buf.WriteByte('\n')
	}
	t.Log(buf)
}

func TestRandomGaugeParser(t *testing.T) {
	cfgs := []any{
		&RandomGaugeParserConfig{Min: 1, Max: 13, MaxRepeat: 10},
		&RandomGaugeParserConfig{Min: 109, Max: 728, MaxRepeat: 3},
		&RandomGaugeParserConfig{Min: 11000109, Max: 21000109, MaxRepeat: 1},
		&RandomGaugeParserConfig{Min: -5, Max: +5, MaxRepeat: 7},
		&RandomGaugeParserConfig{Min: 987, Max: 987, MaxRepeat: 7},
	}
	testRandomParser(t, 73, cfgs)
}

func TestRandomCounterParser(t *testing.T) {
	cfgs := []any{
		&RandomCounterParserConfig{Init: 13, MinInc: 2, MaxInc: 7, MaxRepeat: 3},
		&RandomCounterParserConfig{Init: 1013, MinInc: 2, MaxInc: 3, MaxRepeat: 2},
	}
	testRandomParser(t, 73, cfgs)
}

func TestRandomCategoricalParser(t *testing.T) {
	cfgs := []any{
		&RandomCategoricalParserConfig{
			Choices: []string{
				"A",
				"B",
				"C",
				"D",
				"E",
				"x",
				"y",
				"z",
			},
			MaxRepeat: 3,
			//Seed:      1959,
		},
	}
	testRandomParser(t, 73, cfgs)
}
