// Pseudo-parser to be used as data source in Reference VMI

package parser

// Return a random selection from a list of choices, each selection repeated
// 1..N times:
type RandomCategoricalParser struct {
	// Current value, observing minimal parsing format:
	Val []byte
	// The list of choices:
	choices [][]byte
	// Tne random gauge underlying the selection:
	selector *RandomGaugeParser
}

type RandomCategoricalParserConfig struct {
	// Choices:
	Choices []string `yaml:"choices"`
	// Max repeat count:
	// Max repeat count:
	MaxRepeat int32 `yaml:"max_repeat"`
	// Seed:
	Seed int64 `yaml:"seed"`
}

func DefaultRandomCategoricalParserConfig() *RandomCategoricalParserConfig {
	return &RandomCategoricalParserConfig{}
}

func NewRandomCategoricalParser(cfg *RandomCategoricalParserConfig) *RandomCategoricalParser {
	parser := &RandomCategoricalParser{}
	if len(cfg.Choices) > 0 {
		parser.choices = make([][]byte, len(cfg.Choices))
		for i, choice := range cfg.Choices {
			parser.choices[i] = []byte(choice)
		}
		parser.selector = NewRandomGaugeParser(&RandomGaugeParserConfig{
			Min:       0,
			Max:       int32(len(cfg.Choices) - 1),
			MaxRepeat: cfg.MaxRepeat,
			Seed:      cfg.Seed,
		})
	}
	for i, choice := range cfg.Choices {
		parser.choices[i] = []byte(choice)
	}
	return parser
}

func (parser *RandomCategoricalParser) Parse() error {
	if parser.selector != nil {
		parser.selector.update(false)
		parser.Val = parser.choices[parser.selector.ValInt]
	}
	return nil
}
