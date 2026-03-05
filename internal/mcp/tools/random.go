package tools

import (
	"fmt"
	"sort"
	"strings"
)

func randomByArgs(args map[string]any) *javaRandom {
	seedBase := javaMapString(args)
	var seed int64
	for _, b := range []byte(seedBase) {
		seed = seed*31 + int64(b)
	}
	return newJavaRandom(seed)
}

func javaMapString(args map[string]any) string {
	if len(args) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, javaValueString(args[key])))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func javaValueString(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case map[string]any:
		return javaMapString(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, javaValueString(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprint(typed)
	}
}

type javaRandom struct {
	seed uint64
}

const (
	javaMultiplier = 0x5DEECE66D
	javaAddend     = 0xB
	javaMask       = (1 << 48) - 1
)

func newJavaRandom(seed int64) *javaRandom {
	return &javaRandom{seed: (uint64(seed) ^ javaMultiplier) & javaMask}
}

func (r *javaRandom) next(bits uint) int32 {
	r.seed = (r.seed*javaMultiplier + javaAddend) & javaMask
	return int32(r.seed >> (48 - bits))
}

func (r *javaRandom) NextInt(bound int) int {
	if bound <= 0 {
		return 0
	}
	if bound&(bound-1) == 0 {
		return int((int64(bound) * int64(r.next(31))) >> 31)
	}
	for {
		bits := int(r.next(31))
		val := bits % bound
		if bits-val+(bound-1) >= 0 {
			return val
		}
	}
}

func (r *javaRandom) NextBool() bool {
	return r.next(1) != 0
}
