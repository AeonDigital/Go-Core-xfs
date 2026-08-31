package xstrings

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// FormatSingleValuesToOutput aggregates a slice of strings into a standardized visual output sequence,
// wrapping each individual element within the specified open/close tokens and separating them via a delimiter.
// If the provided strings.Builder pointer is nil, the factory safely initializes a new instance before
// executing high-performance buffer writes. It returns the active builder to support fluent API chaining.
func FormatSingleValuesToOutput(
	open string,
	close string,
	sep string,
	values []string,
	sb *strings.Builder,
) *strings.Builder {
	if sb == nil {
		sb = &strings.Builder{}
	}

	for i, val := range values {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(open)
		sb.WriteString(val)
		sb.WriteString(close)
	}

	return sb
}

// FormatPairValuesToOutput processes a collection of strings in sequential pairs, formatting and merging
// them into a structural visual engine. It encapsulates each part of the pair inside custom open/close boundaries,
// unifies the relationship via unionPair, and joins independent pairs together using the specified sepPair delimiter.
//
// Operational Edge Case:
//   - If the incoming slice contains an odd number of items (or a single element), the trailing orphan element
//     is intercepted and treated as an individual segment, preventing layout breaks or out-of-bounds panics.
func FormatPairValuesToOutput(
	openFirst string,
	closeFirst string,
	unionPair string,
	sepPair string,
	openSecond string,
	closeSecond string,
	values []string,
	sb *strings.Builder,
) *strings.Builder {
	if sb == nil {
		sb = &strings.Builder{}
	}

	for i := 0; i < len(values); i += 2 {
		// Append the custom pair separator if this is not the first pair sequence
		if i > 0 {
			sb.WriteString(sepPair)
		}

		// Step 1: Format the first element of the pair sequence
		sb = FormatSingleValuesToOutput(openFirst, closeFirst, "", []string{values[i]}, sb)

		// Step 2: Check if there is a matching second element to complete the pair contract
		if i+1 < len(values) {
			sb.WriteString(unionPair)
			sb = FormatSingleValuesToOutput(openSecond, closeSecond, "", []string{values[i+1]}, sb)
		}
	}

	return sb
}

// FormatPairsColon formats a slice of sequential key-value string pairs into a standardized,
// single-quote encapsulated metadata string, writing the structured output directly to a strings.Builder.
// If the provided strings.Builder pointer is nil, a new instance will be automatically initialized and returned.
// It applies a consistent layout with single quotes around keys and values, separated by colons,
// and joined by pipes (e.g., 'Key': 'Value' | 'Key2': 'Value2').
func FormatPairsColon(pairs []string, sb *strings.Builder) *strings.Builder {
	return FormatPairValuesToOutput(
		"'", "'", ": ", " | ", "'", "'",
		pairs, sb,
	)
}

// FormatPairsArrow formats a slice of sequential key-value string pairs into a standardized
// mapping string representation, writing the structured output directly to a strings.Builder.
// If the provided strings.Builder pointer is nil, a new instance will be automatically initialized and returned.
// It applies a consistent layout with single quotes around keys, an arrow separator, unquoted values,
// and pairs joined by pipes (e.g., 'Key' => Value | 'Key2' => Value2).
func FormatPairsArrow(pairs []string, sb *strings.Builder) *strings.Builder {
	return FormatPairValuesToOutput(
		"", "", " => ", " | ", "", "",
		pairs, sb,
	)
}

// ParseStructToString converts any struct instance into a structured visual sequence for technical logging.
// It formats the output matching the pattern: <StructName>{"key":"value"}.
// If the struct contains no public (exported) fields, it falls back immediately to <StructName>{}.
// It automatically dereferences pointer arguments to extract and evaluate the underlying struct data structure.
func ParseStructToString(item any) string {
	if item == nil {
		return "<nil>"
	}

	val := reflect.ValueOf(item)

	// Automatically resolve pointer references to extract the underlying values
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	// Terminate early and fallback to default formatting if the target argument is not a struct
	if val.Kind() != reflect.Struct {
		return fmt.Sprintf("%v", item)
	}

	structName := val.Type().Name()
	hasPublicFields := false

	// Scan struct fields sequentially to determine the existence of exported properties
	for i := 0; i < val.NumField(); i++ {
		if val.Type().Field(i).IsExported() {
			hasPublicFields = true
			break
		}
	}

	// Intercept orphan structures lacking public fields to prevent empty brackets or parsing leaks
	if !hasPublicFields {
		return fmt.Sprintf("%s{}", structName)
	}

	// Serialize active public data slices into standard compact JSON payloads
	jsonData, err := json.Marshal(item)
	if err != nil {
		return fmt.Sprintf("%s{error:%s}", structName, err.Error())
	}

	return fmt.Sprintf("%s%s", structName, string(jsonData))
}

// ConvertAllToStrings transforms a variadic sequence of arguments of any underlying type into a structured string slice.
// It preserves the integrity and identity of iterable collections (slices/arrays) by formatting them as single unified
// visual entries matching their type signature (e.g., []Type{...}), preventing structural data leakage into independent index segments.
func ConvertAllToStrings(args ...any) []string {
	var result []string

	for _, arg := range args {
		if arg == nil {
			result = append(result, "")
			continue
		}

		val := reflect.ValueOf(arg)
		kind := val.Kind()

		// If it is a pointer, resolve its underlying element to evaluate its concrete type
		if kind == reflect.Pointer {
			val = val.Elem()
			kind = val.Kind()
		}

		// Intercept complex individual data structures (Structs) to apply the visual log layout format
		if kind == reflect.Struct {
			result = append(result, ParseStructToString(arg))
			continue
		}

		// Evaluate iterable collections (Slices/Arrays), preserving their structural encapsulation
		if (kind == reflect.Slice || kind == reflect.Array) && val.Type().Elem().Kind() != reflect.Uint8 {
			var innerElements []string

			for i := 0; i < val.Len(); i++ {
				element := val.Index(i).Interface()
				elementVal := reflect.ValueOf(element)
				elementKind := elementVal.Kind()

				if elementKind == reflect.Pointer {
					elementVal = elementVal.Elem()
					elementKind = elementVal.Kind()
				}

				// If the inner element is a struct, format it using JSON format without the prefix name
				switch elementKind {
				case reflect.Struct:
					// We reuse ConvertAllToStrings or ParseStructToString logic, stripping the struct name
					// if we prefer pure JSON inside collections, or keeping it depending on the style guide.
					// For a clean block look, we use a specialized approach for inner struct items:
					structStr := ParseStructToString(element)
					if idx := strings.Index(structStr, "{"); idx != -1 {
						structStr = structStr[idx:] // Keeps only the {"key":"value"} payload
					}
					innerElements = append(innerElements, structStr)
				case reflect.Slice, reflect.Array:
					// Recursively handle nested collections (e.g., matrices like [][]int)
					nestedResult := ConvertAllToStrings(element)
					if len(nestedResult) > 0 {
						innerElements = append(innerElements, nestedResult[0])
					}
				default:
					innerElements = append(innerElements, fmt.Sprintf("%v", element))
				}
			}

			// Reconstruct the collection signature string: e.g., []int{1, 2} or []Role{{"type":"Admin"}}
			collectionType := val.Type().String()
			formattedCollection := fmt.Sprintf("%s{%s}", collectionType, strings.Join(innerElements, ","))
			result = append(result, formattedCollection)

		} else {
			// Fallback formatting routine for generic primitives and unhandled systems
			result = append(result, fmt.Sprintf("%v", arg))
		}
	}

	return result
}

// BuildErrorInfo aggregates up to five positional diagnostic variables into a unified,
// scannable metadata string pattern: Field -> A | Given -> B | Expected -> C | Extracted -> D | Reason -> E.
//
// Arguments must adhere strictly to the following structural positions:
//   - index 0: Field (The identifier or key path of the parameter being evaluated)
//   - index 1: Given (The raw input value triggering the exception)
//   - index 2: Expected (The targeted success criteria or standard contract)
//   - index 3: Extracted (The isolated payload snapshot or transformed segment, optional)
//   - index 4: Reason (The underlying logical breakdown explanation, optional)
//
// Trailing unprovided positions or empty strings are safely intercepted and pruned from the final buffer.
func BuildErrorInfo(args ...string) string {
	if len(args) == 0 {
		return ""
	}

	// Internal position mapping definitions
	labels := [5]string{"Field", "Given", "Expected", "Extracted", "Reason"}
	var pairs []string

	// Scan through the arguments up to the maximum managed positional limits
	for i := 0; i < len(args) && i < len(labels); i++ {
		val := args[i]
		if val == "" {
			continue
		}

		pairs = append(pairs, labels[i], val)
	}

	if len(pairs) == 0 {
		return ""
	}

	sb := FormatPairsArrow(pairs, nil)
	return sb.String()
}
