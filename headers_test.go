package stompy

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

//encoded key decoded value
var testEncodeData = map[string]string{
	"astring":             "astring",
	"\\\\":                "\\",
	"\\n":                 "\n",
	"\\c":                 ":",
	"\\\\\\n\\c":          "\\\n:",
	"\\c\\n\\\\":          ":\n\\",
	"\\\\\\c":             "\\:",
	"c\\cc":               "c:c",
	"n\\nn":               "n\nn",
	"test\\cvalue\\ntest": "test:value\ntest",
}

func TestHeaders_encode(t *testing.T) {
	encoder := headerEncoderDecoder{"1.1"}
	for to, from := range testEncodeData {
		fmt.Println("encoding from ", from, "to", to)
		enc := encoder.Encode(from)
		assert.Equal(t, to, enc, "expected encoded value")
	}

}

func TestHeaders_decode(t *testing.T) {
	decoder := headerEncoderDecoder{"1.1"}
	for to, from := range testEncodeData {
		fmt.Println("decoding from ", from, "to", to)
		enc := decoder.Decode(to)
		assert.Equal(t, from, enc, "expected encoded value")
	}
}
