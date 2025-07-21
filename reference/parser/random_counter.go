// Pseudo-parser to be used as data source in Reference VMI

package parser

import (
	"math/rand"
)

// Increment a counter by a random number 1..M, each value repeated 1..N times:
type RandomCounterParser struct {
	// Current value:
	Val uint32
	// Left count for the current value; when it reaches 0, a new value and a
	// new count are generated:
	countLeft int32
	// The updater functions, one for value and the other one for count; they
	// will be established when the parser is built:
	valUpdater, countUpdater func()
}

type RandomCounterParserConfig struct {
	// Init value:
	Init uint32 `yaml:"init"`
	// Range for the increment, when the value changes:
	MinInc uint32 `yaml:"min_inc"`
	MaxInc uint32 `yaml:"max_inc"`
	// Max repeat count:
	MaxRepeat int32 `yaml:"max_repeat"`
	// Seed:
	Seed int64 `yaml:"seed"`
}

func DefaultRandomCounterParserConfig() *RandomCounterParserConfig {
	return &RandomCounterParserConfig{
		Init:      0,
		MinInc:    1,
		MaxInc:    1, // i.e. constant increment
		MaxRepeat: 1,
		Seed:      0, // i.e. no seed
	}
}

func NewRandomCounterParser(cfg *RandomCounterParserConfig) *RandomCounterParser {
	if cfg == nil {
		cfg = DefaultRandomCounterParserConfig()
	}

	parser := &RandomCounterParser{
		Val:       cfg.Init,
		countLeft: 1,
	}

	var int31nFunc = rand.Int31n
	if cfg.Seed > 0 {
		randSrc := rand.New(rand.NewSource(cfg.Seed))
		int31nFunc = randSrc.Int31n
	}

	if cfg.MaxInc > cfg.MinInc {
		n := int32(cfg.MaxInc - cfg.MinInc + 1)
		parser.valUpdater = func() {
			parser.Val += uint32(int31nFunc(n)) + cfg.MinInc
		}
	} else {
		parser.valUpdater = func() {
			parser.Val += cfg.MinInc
		}
	}

	if cfg.MaxRepeat > 1 {
		parser.countUpdater = func() {
			parser.countLeft = int31nFunc(cfg.MaxRepeat)
		}
	}

	return parser
}

func (parser *RandomCounterParser) Parse() error {
	if parser.countLeft > 0 {
		parser.countLeft -= 1
	} else {
		parser.valUpdater()
		if parser.countUpdater != nil {
			parser.countUpdater()
		}
	}
	return nil
}
