// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"crypto/md5"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
)

// NormalizeSqlAndHash normalizes a SQL statement following New Relic Java agent logic
// and returns both the normalized SQL and its MD5 hash.
// This is used for cross-language SQL comparison and query identification.
//
// Normalization rules (SQL Server T-SQL specific):
// - Converts to uppercase
// - Normalizes T-SQL bind variables (@name, @1) to '?'
// - Normalizes JDBC placeholders (?) to '?'
// - Replaces string and numeric literals with '?'
// - Removes comments (single-line and multi-line)
// - Normalizes whitespace (collapses multiple spaces into single space)
// - Normalizes IN clauses with multiple values to IN (?)
func NormalizeSqlAndHash(sql string) (normalizedSQL, hash string) {
	normalizedSQL = NormalizeSql(sql)
	hash = GenerateMD5Hash(normalizedSQL)
	return normalizedSQL, hash
}

// NormalizeSql normalizes a SQL statement based on New Relic Java agent rules.
func NormalizeSql(sql string) string {
	if sql == "" {
		return ""
	}
	// Force uppercase BEFORE normalization starts (matches Java Agent behavior)
	sql = strings.ToUpper(sql)
	sql = normalizeParametersAndLiterals(sql)
	return removeCommentsAndNormalizeWhitespace(sql)
}

// GenerateMD5Hash generates an MD5 hash of the normalized SQL
func GenerateMD5Hash(normalizedSQL string) string {
	hash := md5.Sum([]byte(normalizedSQL))
	return hex.EncodeToString(hash[:])
}

// ExtractNewRelicMetadata extracts nr_apm_guid and nr_service from New Relic query comments
// REQUIRED FORMAT: Values must be enclosed in double quotes to handle commas and special characters
//
// Supported formats:
// 1. APM GUID and Service: /* nr_apm_guid="MTE2MDAzMTl8QVBNfEFQUExJQ0FUSU9OfDI5MjMzNDQwNw", nr_service="order-service" */
// 2. Service GUID variant: /* nr_service_guid="MTE2MDAzMTl8QVBNfEFQUExJQ0FUSU9OfDI5MjMzNDQwNw", nr_service="order-service" */
// 3. Service only: /* nr_service="MyApp-SQLServer, Background Job" */
// 4. Any order: /* nr_service="MyApp", nr_apm_guid="XYZ789" */
// 5. With spaces: /* nr_service = "MyApp" , nr_apm_guid = "ABC" */
// 6. APM GUID only: /* nr_apm_guid="ABC123" */
//
// Returns: (nr_apm_guid, client_name)
func ExtractNewRelicMetadata(sql string) (nrApmGuid, nrService string) {
	// Match nr_apm_guid OR nr_service_guid OR nr_guid with quoted values (spaces around = are optional)
	// Format: nr_apm_guid = "base64_encoded_guid" OR nr_service_guid = "base64_encoded_guid" OR nr_guid = "base64_encoded_guid"
	// Try in order: nr_apm_guid, nr_service_guid, nr_guid (shortest)
	apmGuidRegex := regexp.MustCompile(`nr_apm_guid\s*=\s*"([^"]+)"`)
	serviceGuidRegex := regexp.MustCompile(`nr_service_guid\s*=\s*"([^"]+)"`)
	guidRegex := regexp.MustCompile(`nr_guid\s*=\s*"([^"]+)"`)

	// Match nr_service with quoted values (spaces around = are optional)
	// Format: nr_service = "value with, commas and special chars"
	serviceRegex := regexp.MustCompile(`nr_service\s*=\s*"([^"]+)"`)

	// Extract nr_apm_guid (try standard format first, then variants)
	apmGuidMatch := apmGuidRegex.FindStringSubmatch(sql)
	if len(apmGuidMatch) > 1 {
		nrApmGuid = strings.TrimSpace(apmGuidMatch[1])
	} else {
		// Try nr_service_guid variant
		serviceGuidMatch := serviceGuidRegex.FindStringSubmatch(sql)
		if len(serviceGuidMatch) > 1 {
			nrApmGuid = strings.TrimSpace(serviceGuidMatch[1])
		} else {
			// Try nr_guid (shortest variant)
			guidMatch := guidRegex.FindStringSubmatch(sql)
			if len(guidMatch) > 1 {
				nrApmGuid = strings.TrimSpace(guidMatch[1])
			}
		}
	}

	// Extract nr_service
	serviceMatch := serviceRegex.FindStringSubmatch(sql)
	if len(serviceMatch) > 1 {
		nrService = strings.TrimSpace(serviceMatch[1])
	}

	return nrApmGuid, nrService
}

// sqlNormalizerState holds state during SQL normalization
type sqlNormalizerState struct {
	sql               string
	length            int
	idx               int
	lastWasWhitespace bool
}

func newSqlNormalizerState(sql string) *sqlNormalizerState {
	return &sqlNormalizerState{
		sql:               sql,
		length:            len(sql),
		idx:               0,
		lastWasWhitespace: true, // Start as true to trim leading whitespace
	}
}

func (s *sqlNormalizerState) hasMore() bool {
	return s.idx < s.length
}

func (s *sqlNormalizerState) hasNext() bool {
	return s.idx+1 < s.length
}

func (s *sqlNormalizerState) current() byte {
	return s.sql[s.idx]
}

func (s *sqlNormalizerState) peek() byte {
	return s.sql[s.idx+1]
}

func (s *sqlNormalizerState) advance() {
	s.idx++
}

func (s *sqlNormalizerState) advanceBy(count int) {
	s.idx += count
}

// normalizeParametersAndLiterals normalizes all parameter placeholders and literals
func normalizeParametersAndLiterals(sql string) string {
	if sql == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(sql))
	state := newSqlNormalizerState(sql)

	for state.hasMore() {
		current := state.current()

		if current == '\'' {
			// Replace string literals with ?
			skipStringLiteral(state)
			result.WriteByte('?')
		} else if current == '(' {
			// Check for IN clause with multiple values/placeholders
			if isPrecededByIn(&result) {
				inClause := tryNormalizeInClause(state)
				result.WriteString(inClause)
			} else {
				result.WriteByte('(')
				state.advance()
			}
		} else if isNumericLiteral(state) {
			// Numeric literals
			skipNumericLiteral(state)
			result.WriteByte('?')
		} else if isPlaceholder(state) {
			// Any placeholder type (T-SQL @param or JDBC ?) --> ?
			skipPlaceholder(state)
			result.WriteByte('?')
		} else {
			// Just append anything else
			result.WriteByte(current)
			state.advance()
		}
	}

	return result.String()
}

// isPrecededByIn checks if the result is preceded by "IN"
func isPrecededByIn(result *strings.Builder) bool {
	str := result.String()
	if len(str) < 2 {
		return false
	}

	// Scan backwards, skipping whitespace
	idx := len(str) - 1
	for idx >= 0 && unicode.IsSpace(rune(str[idx])) {
		idx--
	}

	// Check if we have at least "IN" (2 characters)
	if idx < 1 {
		return false
	}

	// Check for "IN" - scanning backwards we see 'N' first, then 'I'
	if str[idx] == 'N' && str[idx-1] == 'I' {
		// Make sure "IN" is a complete token, not part of a larger word like "WITHIN"
		return idx < 2 || !isIdentifierChar(rune(str[idx-2]))
	}

	return false
}

// isIdentifierChar checks if a character is valid in an identifier
func isIdentifierChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_'
}

// isPlaceholder checks if current position is a parameter placeholder
// Supports multiple database placeholder formats:
// - JDBC: ?
// - T-SQL: @paramname or @1
// - Oracle: :paramname or :1
// - PostgreSQL: $1, $2, etc.
// - Python: %(name)s
func isPlaceholder(state *sqlNormalizerState) bool {
	current := state.current()

	// JDBC-style placeholder
	if current == '?' {
		return true
	}

	// T-SQL named parameter: @paramname
	if current == '@' {
		// Make sure it's not just a lone @
		if !state.hasNext() {
			return false
		}

		next := state.peek()
		// @ followed by letter, digit, or underscore is a parameter
		return unicode.IsLetter(rune(next)) || unicode.IsDigit(rune(next)) || next == '_'
	}

	// Oracle bind variable: :paramname or :1
	if current == ':' && state.hasNext() {
		next := state.peek()
		return unicode.IsLetter(rune(next)) || unicode.IsDigit(rune(next)) || next == '_'
	}

	// PostgreSQL positional parameter: $1, $2, etc.
	if current == '$' && state.hasNext() {
		next := state.peek()
		return unicode.IsDigit(rune(next))
	}

	// Python-style placeholder: %(name)s
	if current == '%' && state.hasNext() && state.peek() == '(' {
		return true
	}

	return false
}

// skipPlaceholder skips over a placeholder
func skipPlaceholder(state *sqlNormalizerState) {
	current := state.current()

	if current == '?' {
		state.advance()
		return
	}

	// T-SQL parameter: @name or @123
	if current == '@' {
		state.advance()
		// Skip the parameter name
		for state.hasMore() {
			c := state.current()
			if unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || c == '_' {
				state.advance()
			} else {
				break
			}
		}
		return
	}

	// Oracle bind variable: :name or :1
	if current == ':' {
		state.advance()
		// Skip the parameter name
		for state.hasMore() {
			c := state.current()
			if unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || c == '_' {
				state.advance()
			} else {
				break
			}
		}
		return
	}

	// PostgreSQL positional parameter: $1, $2, etc.
	if current == '$' {
		state.advance()
		// Skip digits
		for state.hasMore() && unicode.IsDigit(rune(state.current())) {
			state.advance()
		}
		return
	}

	// Python-style placeholder: %(name)s
	if current == '%' && state.hasNext() && state.peek() == '(' {
		state.advanceBy(2) // Skip %(
		// Skip until closing )
		for state.hasMore() && state.current() != ')' {
			state.advance()
		}
		if state.hasMore() && state.current() == ')' {
			state.advance() // Skip )
		}
		// Skip type specifier (s, d, etc.) if present
		if state.hasMore() && unicode.IsLetter(rune(state.current())) {
			state.advance()
		}
	}
}

// isNumericLiteral checks if current position is a numeric literal
func isNumericLiteral(state *sqlNormalizerState) bool {
	current := state.current()

	// Must start with a digit or minus sign (for negative numbers)
	if !unicode.IsDigit(rune(current)) && current != '-' && current != '+' {
		return false
	}

	// If it's a sign, next must be a digit
	if current == '-' || current == '+' {
		if !state.hasNext() {
			return false
		}
		next := state.peek()
		if !unicode.IsDigit(rune(next)) {
			return false
		}
	}

	// Make sure it's not part of an identifier (e.g., table1, _2column)
	// Check if preceded by identifier character
	if state.idx > 0 {
		prev := state.sql[state.idx-1]
		// If preceded by letter, digit, underscore, or backtick, it's part of identifier
		if unicode.IsLetter(rune(prev)) || prev == '_' || prev == '`' {
			return false
		}
	}

	return true
}

// skipNumericLiteral skips over a numeric literal (including scientific notation)
func skipNumericLiteral(state *sqlNormalizerState) {
	// Skip optional sign
	if state.current() == '-' || state.current() == '+' {
		state.advance()
	}

	// Skip digits before decimal point
	for state.hasMore() && unicode.IsDigit(rune(state.current())) {
		state.advance()
	}

	// Skip decimal point and digits after
	if state.hasMore() && state.current() == '.' {
		state.advance()
		for state.hasMore() && unicode.IsDigit(rune(state.current())) {
			state.advance()
		}
	}

	// Skip scientific notation (e.g., 1.5E6, 2e-3)
	if state.hasMore() && (state.current() == 'E' || state.current() == 'e') {
		state.advance()
		// Skip optional sign in exponent
		if state.hasMore() && (state.current() == '+' || state.current() == '-') {
			state.advance()
		}
		// Skip exponent digits
		for state.hasMore() && unicode.IsDigit(rune(state.current())) {
			state.advance()
		}
	}
}

// skipStringLiteral skips over a string literal, handling escaped quotes
func skipStringLiteral(state *sqlNormalizerState) {
	state.advance() // Skip the opening quote

	for state.hasMore() {
		c := state.current()

		switch c {
		case '\'':
			// Check for escaped quote ''
			if !state.hasNext() || state.peek() != '\'' {
				state.advance() // Skip closing quote
				return
			}
			state.advanceBy(2) // Skip both quotes
		case '\\':
			// Handle backslash escaping (MySQL, PostgreSQL)
			state.advance()
			if state.hasMore() {
				state.advance()
			}
		default:
			state.advance()
		}
	}
}

// tryNormalizeInClause attempts to normalize an IN clause
func tryNormalizeInClause(state *sqlNormalizerState) string {
	startIdx := state.idx

	// Skip opening parenthesis
	if state.current() != '(' {
		return string(state.sql[startIdx])
	}
	state.advance()

	hasMultipleValues := false
	valueCount := 0

	// Scan through the clause
	for state.hasMore() {
		current := state.current()

		if current == ')' {
			state.advance()
			break
		} else if current == ',' {
			hasMultipleValues = true
			state.advance()
		} else if current == '\'' {
			skipStringLiteral(state)
			valueCount++
		} else if unicode.IsSpace(rune(current)) {
			state.advance()
		} else if isNumericLiteral(state) {
			skipNumericLiteral(state)
			valueCount++
		} else if isPlaceholder(state) {
			skipPlaceholder(state)
			valueCount++
		} else {
			// Not a simple IN clause, return original
			length := state.idx - startIdx
			return string(state.sql[startIdx : startIdx+length])
		}
	}

	// If multiple values or placeholders found, normalize to IN (?)
	if hasMultipleValues || valueCount > 1 {
		return "(?)"
	}

	// Single value, return IN (?)
	if valueCount == 1 {
		return "(?)"
	}

	// Empty or invalid, return what we consumed
	length := state.idx - startIdx
	if length > 0 {
		return string(state.sql[startIdx : startIdx+length])
	}
	return "()"
}

// processStringLiteral handles string literals in the comment removal phase
// Dead code in practice since string literals are replaced in pass 1,
// but included for defensive completeness matching Oracle reference behavior.
func processStringLiteral(result *strings.Builder, state *sqlNormalizerState) {
	result.WriteByte(state.current())
	state.lastWasWhitespace = false
	state.advance()

	for state.hasMore() {
		c := state.current()
		result.WriteByte(c)

		if c == '\'' {
			//nolint:revive // early-return pattern is more readable here
			if state.hasNext() && state.peek() == '\'' {
				result.WriteByte('\'')
				state.advanceBy(2)
			} else {
				state.advance()
				break
			}
		} else {
			state.advance()
		}
	}
	state.lastWasWhitespace = false
}

// isMultilineCommentStart checks if current position is start of multiline comment
func isMultilineCommentStart(state *sqlNormalizerState) bool {
	return state.current() == '/' && state.hasNext() && state.peek() == '*'
}

// isSingleLineCommentStart checks if current position is start of single-line comment
func isSingleLineCommentStart(state *sqlNormalizerState) bool {
	return state.current() == '-' && state.hasNext() && state.peek() == '-'
}

// skipToEndOfLine skips characters until end of line
func skipToEndOfLine(state *sqlNormalizerState) {
	for state.hasMore() && state.current() != '\n' && state.current() != '\r' {
		state.advance()
	}
	for state.hasMore() && (state.current() == '\n' || state.current() == '\r') {
		state.advance()
	}
}

// processWhitespace handles whitespace normalization
func processWhitespace(result *strings.Builder, state *sqlNormalizerState) {
	if !state.lastWasWhitespace && result.Len() > 0 {
		result.WriteByte(' ')
		state.lastWasWhitespace = true
	}
	state.advance()
}

// processRegularCharacter handles regular characters
func processRegularCharacter(result *strings.Builder, state *sqlNormalizerState) {
	result.WriteByte(state.current())
	state.lastWasWhitespace = false
	state.advance()
}

// removeCommentsAndNormalizeWhitespace strips all comments (single-line --, multi-line /* */, hash #)
// and normalizes whitespace (collapses multiple spaces into single space).
// Prefix comments (before any SQL content) are replaced with '?' to match NR APM agent behavior.
// Inline comments (after SQL content has started) are silently removed.
// processStringLiteral is dead code in practice since literals are replaced in pass 1,
// but included for defensive completeness matching Oracle reference behavior.
func removeCommentsAndNormalizeWhitespace(sql string) string {
	var result strings.Builder
	result.Grow(len(sql))
	state := newSqlNormalizerState(sql)
	seenSQLContent := false // tracks whether any actual SQL content has been written

	for state.hasMore() {
		current := state.current()

		switch {
		case current == '\'':
			// Dead code in practice: string literals were already replaced in pass 1
			seenSQLContent = true
			processStringLiteral(&result, state)
		case isMultilineCommentStart(state):
			if !seenSQLContent {
				// Prefix comment: replace with ? (matches NR APM agent behavior)
				skipMultiLineComment(state)
				result.WriteByte('?')
				state.lastWasWhitespace = false
				seenSQLContent = true
			} else {
				// Inline comment: silently remove
				skipMultiLineComment(state)
			}
		case isSingleLineCommentStart(state):
			if !seenSQLContent {
				// Prefix comment: replace with ?
				state.advanceBy(2) // skip --
				skipToEndOfLine(state)
				result.WriteByte('?')
				state.lastWasWhitespace = false
				seenSQLContent = true
			} else {
				state.advanceBy(2) // skip --
				skipToEndOfLine(state)
			}
		case current == '#':
			if !seenSQLContent {
				// Prefix comment: replace with ?
				state.advance() // skip #
				skipToEndOfLine(state)
				result.WriteByte('?')
				state.lastWasWhitespace = false
				seenSQLContent = true
			} else {
				state.advance() // skip #
				skipToEndOfLine(state)
			}
		case unicode.IsSpace(rune(current)):
			processWhitespace(&result, state)
		default:
			seenSQLContent = true
			processRegularCharacter(&result, state)
		}
	}

	return strings.TrimSpace(result.String())
}

// skipMultiLineComment skips a multi-line comment (/* comment */)
func skipMultiLineComment(state *sqlNormalizerState) {
	state.advanceBy(2) // skip /*
	for state.idx < state.length-1 {
		if state.current() == '*' && state.peek() == '/' {
			state.advanceBy(2)
			return
		}
		state.advance()
	}
	// Handle unclosed comment
	if state.hasMore() {
		state.advance()
	}
}
