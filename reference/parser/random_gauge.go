// Pseudo-parser to be used as data source in Reference VMI

package parser

import (
	"bytes"
	"fmt"
	"math/rand"
)

// Return a random number min..max, each value repeated 1..N times:
type RandomGaugeParser struct {
	// Current value, observing minimal parsing format:
	ValBytes []byte
	// Raw current value:
	ValInt int32
	// Left count for the current value; when it reaches 0, a new value and a
	// new count are generated:
	countLeft int32
	// The updater functions, one for value and the other one for count; they
	// will be established when the parser is built:
	valUpdater, countUpdater func()
	// Underlying buffer for the value:
	buf *bytes.Buffer
}

type RandomGaugeParserConfig struct {
	// Range for the value:
	Min int32 `yaml:"min"`
	Max int32 `yaml:"max"`
	// Max repeat count:
	MaxRepeat int32 `yaml:"max_repeat"`
	// Seed:
	Seed int64 `yaml:"seed"`
}

func DefaultRandomGaugeParserConfig() *RandomGaugeParserConfig {
	return &RandomGaugeParserConfig{
		Min:       0,
		Max:       -1, // i.e. no max
		MaxRepeat: 1,
		Seed:      0, // i.e. no seed
	}
}

func NewRandomGaugeParser(cfg *RandomGaugeParserConfig) *RandomGaugeParser {
	if cfg == nil {
		cfg = DefaultRandomGaugeParserConfig()
	}

	parser := &RandomGaugeParser{
		countLeft: 0,
		buf:       &bytes.Buffer{},
	}

	var int31Func, int31nFunc = rand.Int31, rand.Int31n
	if cfg.Seed > 0 {
		randSrc := rand.New(rand.NewSource(cfg.Seed))
		int31Func, int31nFunc = randSrc.Int31, randSrc.Int31n
	}

	if n := cfg.Max - cfg.Min; n > 0 {
		parser.valUpdater = func() {
			parser.ValInt = int31nFunc(n+1) + cfg.Min
		}
	} else if n < 0 {
		parser.valUpdater = func() {
			parser.ValInt = int31Func()
		}
	} else {
		parser.ValInt = cfg.Min
		fmt.Fprintf(parser.buf, "%d", parser.ValInt)
		parser.ValBytes = parser.buf.Bytes()
	}

	if parser.valUpdater != nil && cfg.MaxRepeat > 1 {
		parser.countUpdater = func() {
			parser.countLeft = int31nFunc(cfg.MaxRepeat)
		}
	}

	return parser
}

func (parser *RandomGaugeParser) update(full bool) {
	if parser.valUpdater == nil {
		return
	}
	if parser.countLeft > 0 {
		parser.countLeft -= 1
	} else {
		parser.valUpdater()
		if full {
			parser.buf.Reset()
			fmt.Fprintf(parser.buf, "%d", parser.ValInt)
			parser.ValBytes = parser.buf.Bytes()
		}
		if parser.countUpdater != nil {
			parser.countUpdater()
		}
	}
}

func (parser *RandomGaugeParser) Parse() error {
	parser.update(true)
	return nil
}
