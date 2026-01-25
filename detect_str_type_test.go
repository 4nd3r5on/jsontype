package jsontype_test

import (
	"math/rand"
	"testing"
	"unicode/utf8"

	"github.com/4nd3r5on/jsontype"
)

func Test_detectBase64_randomPools(t *testing.T) {
	const (
		samplesPositive = 1000
		samplesNegative = 300
	)

	rng := rand.New(rand.NewSource(1))

	genUTF8 := func(r *rand.Rand, minVal, maxVal int) []byte {
		n := minVal + r.Intn(maxVal-minVal+1)
		rs := make([]rune, n)
		for i := range rs {
			rs[i] = rune(32 + r.Intn(95)) // printable ASCII
		}
		return []byte(string(rs))
	}

	genUnicodeGarbage := func(r *rand.Rand, minVal, maxVal int) string {
		n := minVal + r.Intn(maxVal-minVal+1)
		rs := make([]rune, n)
		for i := range rs {
			// mix of non-ASCII unicode planes
			switch r.Intn(3) {
			case 0:
				rs[i] = rune(0x0400 + r.Intn(0x04FF-0x0400)) // Cyrillic
			case 1:
				rs[i] = rune(0x4E00 + r.Intn(0x9FFF-0x4E00)) // CJK
			default:
				rs[i] = rune(0x1F300 + r.Intn(0x1F5FF-0x1F300)) // emoji
			}
		}
		return string(rs)
	}

	encByType := map[jsontype.DetectedType]jsontype.Base64Encoding{}
	for _, v := range jsontype.Base64Variants {
		encByType[v.Type] = v.Encoding
	}

	// Positive property: detected ⇒ decodable ⇒ UTF-8
	for _, variant := range jsontype.Base64Variants {
		t.Run(string(variant.Type), func(t *testing.T) {
			for range samplesPositive {
				raw := genUTF8(rng, 8, 128)
				encoded := variant.Encoding.EncodeToString(raw)

				gotType, ok := jsontype.DetectBase64(encoded)
				if !ok {
					t.Fatalf("not detected: %q", encoded)
				}

				enc := encByType[gotType]
				decoded, err := enc.DecodeString(encoded)
				if err != nil {
					t.Fatalf("detected %s but failed to decode: %q", gotType, encoded)
				}
				if !utf8.Valid(decoded) {
					t.Fatalf("decoded invalid UTF-8: %q", encoded)
				}
			}
		})
	}

	// Negative property: random unicode must NOT detect
	t.Run("unicode-garbage", func(t *testing.T) {
		for range samplesNegative {
			s := genUnicodeGarbage(rng, 8, 64)
			if typ, ok := jsontype.DetectBase64(s); ok {
				t.Fatalf("false positive (%s): %q", typ, s)
			}
		}
	})
}
