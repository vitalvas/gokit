package wirefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCloudflareComparisonOperators tests comparison operators from Cloudflare docs.
// Reference: https://developers.cloudflare.com/ruleset-engine/rules-language/operators/
func TestCloudflareComparisonOperators(t *testing.T) {
	schema := NewSchema().
		AddField("http.request.uri.path", TypeString).
		AddField("http.host", TypeString).
		AddField("ip.src", TypeIP).
		AddField("cf.waf.score", TypeInt).
		AddField("http.request.method", TypeString)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		// From: https://developers.cloudflare.com/ruleset-engine/rules-language/operators/
		// Original: http.request.uri.path eq "/articles/2008/"
		{
			name:       "equal_operator",
			expression: `http.request.uri.path == "/articles/2008/"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2008/"), true},
				{"different_value", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2007/"), false},
				{"empty_value", NewExecutionContext().SetStringField("http.request.uri.path", ""), false},
			},
		},
		// Original: ip.src ne 203.0.113.0
		{
			name:       "not_equal_operator",
			expression: `ip.src != 203.0.113.0`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"different_value", NewExecutionContext().SetIPField("ip.src", "192.168.1.1"), true},
				{"same_value", NewExecutionContext().SetIPField("ip.src", "203.0.113.0"), false},
			},
		},
		// Original: cf.waf.score lt 10
		{
			name:       "less_than_operator",
			expression: `cf.waf.score < 10`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"below", NewExecutionContext().SetIntField("cf.waf.score", 5), true},
				{"equal", NewExecutionContext().SetIntField("cf.waf.score", 10), false},
				{"above", NewExecutionContext().SetIntField("cf.waf.score", 15), false},
			},
		},
		// Original: cf.waf.score le 20
		{
			name:       "less_equal_operator",
			expression: `cf.waf.score <= 20`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"below", NewExecutionContext().SetIntField("cf.waf.score", 15), true},
				{"equal", NewExecutionContext().SetIntField("cf.waf.score", 20), true},
				{"above", NewExecutionContext().SetIntField("cf.waf.score", 25), false},
			},
		},
		// Original: cf.waf.score gt 25
		{
			name:       "greater_than_operator",
			expression: `cf.waf.score > 25`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"above", NewExecutionContext().SetIntField("cf.waf.score", 30), true},
				{"equal", NewExecutionContext().SetIntField("cf.waf.score", 25), false},
				{"below", NewExecutionContext().SetIntField("cf.waf.score", 20), false},
			},
		},
		// Original: cf.waf.score ge 60
		{
			name:       "greater_equal_operator",
			expression: `cf.waf.score >= 60`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"above", NewExecutionContext().SetIntField("cf.waf.score", 75), true},
				{"equal", NewExecutionContext().SetIntField("cf.waf.score", 60), true},
				{"below", NewExecutionContext().SetIntField("cf.waf.score", 55), false},
			},
		},
		// Original: http.request.uri.path contains "/articles/"
		{
			name:       "contains_operator",
			expression: `http.request.uri.path contains "/articles/"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"found_middle", NewExecutionContext().SetStringField("http.request.uri.path", "/blog/articles/2024/"), true},
				{"found_start", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/latest"), true},
				{"not_found", NewExecutionContext().SetStringField("http.request.uri.path", "/blog/posts/2024/"), false},
			},
		},
		// Original: http.request.uri.path wildcard "/articles/*"
		{
			name:       "wildcard_operator",
			expression: `http.request.uri.path wildcard "/articles/*"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match_lowercase", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2024"), true},
				{"match_uppercase", NewExecutionContext().SetStringField("http.request.uri.path", "/ARTICLES/2024"), true},
				{"no_match", NewExecutionContext().SetStringField("http.request.uri.path", "/blog/posts"), false},
			},
		},
		// Original: http.request.uri.path strict wildcard "/AdminTeam/*"
		{
			name:       "strict_wildcard_operator",
			expression: `http.request.uri.path strict wildcard "/AdminTeam/*"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"exact_case", NewExecutionContext().SetStringField("http.request.uri.path", "/AdminTeam/dashboard"), true},
				{"wrong_case", NewExecutionContext().SetStringField("http.request.uri.path", "/adminteam/dashboard"), false},
				{"different_path", NewExecutionContext().SetStringField("http.request.uri.path", "/UserTeam/dashboard"), false},
			},
		},
		// Original: http.request.uri.path matches "^/articles/200[7-8]/$"
		{
			name:       "matches_operator",
			expression: `http.request.uri.path matches "^/articles/200[7-8]/$"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match_2007", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2007/"), true},
				{"match_2008", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2008/"), true},
				{"no_match_2009", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2009/"), false},
			},
		},
		// Original: ip.src in { 203.0.113.0 203.0.113.1 }
		{
			name:       "in_set_operator",
			expression: `ip.src in {203.0.113.0, 203.0.113.1}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"first_ip", NewExecutionContext().SetIPField("ip.src", "203.0.113.0"), true},
				{"second_ip", NewExecutionContext().SetIPField("ip.src", "203.0.113.1"), true},
				{"outside", NewExecutionContext().SetIPField("ip.src", "192.168.1.1"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareLogicalOperators tests all logical operators from Cloudflare docs.
func TestCloudflareLogicalOperators(t *testing.T) {
	schema := NewSchema().
		AddField("http.host", TypeString).
		AddField("ip.src", TypeIP).
		AddField("cf.edge.server_port", TypeInt)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "not_word_operator",
			expression: `not (http.host == "www.cloudflare.com")`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"inner_false", NewExecutionContext().SetStringField("http.host", "www.example.com"), true},
				{"inner_true", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com"), false},
				{"empty_value", NewExecutionContext().SetStringField("http.host", ""), true},
			},
		},
		{
			name:       "not_symbol_operator",
			expression: `!(http.host == "www.cloudflare.com")`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"inner_false", NewExecutionContext().SetStringField("http.host", "www.example.com"), true},
				{"inner_true", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com"), false},
				{"empty_value", NewExecutionContext().SetStringField("http.host", ""), true},
			},
		},
		{
			name:       "and_word_operator",
			expression: `http.host == "www.cloudflare.com" and ip.src in "203.0.113.0/24"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"both_true", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com").SetIPField("ip.src", "203.0.113.50"), true},
				{"first_false", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIPField("ip.src", "203.0.113.50"), false},
				{"second_false", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com").SetIPField("ip.src", "192.168.1.1"), false},
				{"both_false", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIPField("ip.src", "192.168.1.1"), false},
			},
		},
		{
			name:       "and_symbol_operator",
			expression: `http.host == "www.cloudflare.com" && ip.src in "203.0.113.0/24"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"both_true", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com").SetIPField("ip.src", "203.0.113.50"), true},
				{"first_false", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIPField("ip.src", "203.0.113.50"), false},
				{"both_false", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIPField("ip.src", "192.168.1.1"), false},
			},
		},
		{
			name:       "xor_word_operator",
			expression: `http.host == "www.cloudflare.com" xor ip.src in "203.0.113.0/24"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"first_true_only", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com").SetIPField("ip.src", "192.168.1.1"), true},
				{"second_true_only", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIPField("ip.src", "203.0.113.50"), true},
				{"both_true", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com").SetIPField("ip.src", "203.0.113.50"), false},
				{"both_false", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIPField("ip.src", "192.168.1.1"), false},
			},
		},
		{
			name:       "or_word_operator",
			expression: `http.host == "www.cloudflare.com" or ip.src in "203.0.113.0/24"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"both_true", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com").SetIPField("ip.src", "203.0.113.50"), true},
				{"first_true", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com").SetIPField("ip.src", "192.168.1.1"), true},
				{"second_true", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIPField("ip.src", "203.0.113.50"), true},
				{"both_false", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIPField("ip.src", "192.168.1.1"), false},
			},
		},
		{
			name:       "or_symbol_operator",
			expression: `http.host == "www.cloudflare.com" || ip.src in "203.0.113.0/24"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"both_true", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com").SetIPField("ip.src", "203.0.113.50"), true},
				{"first_true", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com").SetIPField("ip.src", "192.168.1.1"), true},
				{"both_false", NewExecutionContext().SetStringField("http.host", "www.other.com").SetIPField("ip.src", "10.0.0.1"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareCompoundExpressions tests compound expressions from Cloudflare docs.
func TestCloudflareCompoundExpressions(t *testing.T) {
	schema := NewSchema().
		AddField("http.host", TypeString).
		AddField("http.request.uri.path", TypeString).
		AddField("cf.edge.server_port", TypeInt).
		AddField("http.request.method", TypeString)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "domain_non_standard_port",
			expression: `http.host == "www.example.com" and not cf.edge.server_port in {80, 443}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"non_standard_port", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIntField("cf.edge.server_port", 8080), true},
				{"standard_port_80", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIntField("cf.edge.server_port", 80), false},
				{"standard_port_443", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIntField("cf.edge.server_port", 443), false},
				{"wrong_host", NewExecutionContext().SetStringField("http.host", "other.com").SetIntField("cf.edge.server_port", 8080), false},
			},
		},
		{
			name:       "autodiscover_regex",
			expression: `http.request.uri.path matches "/autodiscover\\.(xml|src)$"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"xml_match", NewExecutionContext().SetStringField("http.request.uri.path", "/autodiscover.xml"), true},
				{"src_match", NewExecutionContext().SetStringField("http.request.uri.path", "/autodiscover.src"), true},
				{"no_match", NewExecutionContext().SetStringField("http.request.uri.path", "/autodiscover.txt"), false},
			},
		},
		{
			name:       "grouping_precedence",
			expression: `(http.host == "www.example.com" or http.host == "api.example.com") and http.request.method == "POST"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"www_post", NewExecutionContext().SetStringField("http.host", "www.example.com").SetStringField("http.request.method", "POST"), true},
				{"api_post", NewExecutionContext().SetStringField("http.host", "api.example.com").SetStringField("http.request.method", "POST"), true},
				{"www_get", NewExecutionContext().SetStringField("http.host", "www.example.com").SetStringField("http.request.method", "GET"), false},
				{"other_post", NewExecutionContext().SetStringField("http.host", "other.example.com").SetStringField("http.request.method", "POST"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareValueTypes tests value types from Cloudflare docs.
func TestCloudflareValueTypes(t *testing.T) {
	schema := NewSchema().
		AddField("http.host", TypeString).
		AddField("http.request.uri.path", TypeString).
		AddField("http.status", TypeInt).
		AddField("ip.src", TypeIP).
		AddField("cf.bot_management.verified_bot", TypeBool)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "string_equality",
			expression: `http.host == "example.com"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.host", "example.com"), true},
				{"different", NewExecutionContext().SetStringField("http.host", "other.com"), false},
				{"empty", NewExecutionContext().SetStringField("http.host", ""), false},
			},
		},
		{
			name:       "raw_string_backslashes",
			expression: `http.request.uri.path == r"path\to\file"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.request.uri.path", `path\to\file`), true},
				{"no_backslashes", NewExecutionContext().SetStringField("http.request.uri.path", "path/to/file"), false},
				{"partial", NewExecutionContext().SetStringField("http.request.uri.path", `path\to`), false},
			},
		},
		{
			name:       "raw_string_regex",
			expression: `http.request.uri.path matches r"^\d+$"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"digits_only", NewExecutionContext().SetStringField("http.request.uri.path", "12345"), true},
				{"mixed", NewExecutionContext().SetStringField("http.request.uri.path", "123abc"), false},
				{"letters_only", NewExecutionContext().SetStringField("http.request.uri.path", "abc"), false},
			},
		},
		{
			name:       "integer_equality",
			expression: `http.status == 200`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetIntField("http.status", 200), true},
				{"different", NewExecutionContext().SetIntField("http.status", 404), false},
				{"zero", NewExecutionContext().SetIntField("http.status", 0), false},
			},
		},
		{
			name:       "integer_range",
			expression: `http.status in {200..299}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"middle", NewExecutionContext().SetIntField("http.status", 250), true},
				{"start", NewExecutionContext().SetIntField("http.status", 200), true},
				{"end", NewExecutionContext().SetIntField("http.status", 299), true},
				{"below", NewExecutionContext().SetIntField("http.status", 199), false},
				{"above", NewExecutionContext().SetIntField("http.status", 300), false},
			},
		},
		{
			name:       "ip_in_cidr_string",
			expression: `ip.src in "192.0.2.0/24"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"inside", NewExecutionContext().SetIPField("ip.src", "192.0.2.100"), true},
				{"first", NewExecutionContext().SetIPField("ip.src", "192.0.2.1"), true},
				{"outside", NewExecutionContext().SetIPField("ip.src", "192.0.3.1"), false},
			},
		},
		{
			name:       "ip_in_cidr_native",
			expression: `ip.src in {192.0.2.0/24}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"inside", NewExecutionContext().SetIPField("ip.src", "192.0.2.100"), true},
				{"first", NewExecutionContext().SetIPField("ip.src", "192.0.2.1"), true},
				{"outside", NewExecutionContext().SetIPField("ip.src", "192.0.3.1"), false},
			},
		},
		{
			name:       "ip_in_multiple_cidrs",
			expression: `ip.src in {192.168.0.0/24, 10.0.0.0/8}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"in_first_cidr", NewExecutionContext().SetIPField("ip.src", "192.168.0.50"), true},
				{"in_second_cidr", NewExecutionContext().SetIPField("ip.src", "10.20.30.40"), true},
				{"outside_both", NewExecutionContext().SetIPField("ip.src", "172.16.0.1"), false},
			},
		},
		{
			name:       "ip_in_mixed_ips_and_cidrs",
			expression: `ip.src in {192.168.1.1, 10.0.0.0/8, 172.16.5.5}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"exact_ip_match", NewExecutionContext().SetIPField("ip.src", "192.168.1.1"), true},
				{"in_cidr", NewExecutionContext().SetIPField("ip.src", "10.50.100.200"), true},
				{"second_exact_ip", NewExecutionContext().SetIPField("ip.src", "172.16.5.5"), true},
				{"no_match", NewExecutionContext().SetIPField("ip.src", "8.8.8.8"), false},
			},
		},
		{
			name:       "ipv6_in_cidr_native",
			expression: `ip.src in {2001:db8::/32}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"inside", NewExecutionContext().SetIPField("ip.src", "2001:db8::1"), true},
				{"outside", NewExecutionContext().SetIPField("ip.src", "2001:db9::1"), false},
			},
		},
		{
			name:       "boolean_field",
			expression: `cf.bot_management.verified_bot`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"true", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", true), true},
				{"false", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", false), false},
			},
		},
		{
			name:       "boolean_negation",
			expression: `not cf.bot_management.verified_bot`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"negate_false", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", false), true},
				{"negate_true", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", true), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareArraysAndMaps tests array and map access from Cloudflare docs.
func TestCloudflareArraysAndMaps(t *testing.T) {
	schema := NewSchema().
		AddField("http.request.headers", TypeMap).
		AddField("http.request.headers.names", TypeArray).
		AddField("http.request.uri.args", TypeMap).
		AddField("tags", TypeArray)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "array_index_access",
			expression: `tags[0] == "important"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match_first", NewExecutionContext().SetArrayField("tags", []string{"important", "urgent", "review"}), true},
				{"wrong_value", NewExecutionContext().SetArrayField("tags", []string{"other", "urgent", "review"}), false},
				{"empty_array", NewExecutionContext().SetArrayField("tags", []string{}), false},
			},
		},
		{
			name:       "map_key_access",
			expression: `http.request.headers["content-type"] == "application/json"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetMapField("http.request.headers", map[string]string{"content-type": "application/json"}), true},
				{"wrong_value", NewExecutionContext().SetMapField("http.request.headers", map[string]string{"content-type": "text/html"}), false},
				{"missing_key", NewExecutionContext().SetMapField("http.request.headers", map[string]string{"accept": "text/html"}), false},
			},
		},
		{
			name:       "array_unpack_contains",
			expression: `tags[*] contains "admin"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"found", NewExecutionContext().SetArrayField("tags", []string{"user", "admin", "reviewer"}), true},
				{"not_found", NewExecutionContext().SetArrayField("tags", []string{"user", "guest", "reviewer"}), false},
				{"empty_array", NewExecutionContext().SetArrayField("tags", []string{}), false},
			},
		},
		{
			name:       "array_unpack_equals",
			expression: `tags[*] == "admin"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"found", NewExecutionContext().SetArrayField("tags", []string{"user", "admin", "reviewer"}), true},
				{"not_found", NewExecutionContext().SetArrayField("tags", []string{"user", "guest", "reviewer"}), false},
				{"partial_match", NewExecutionContext().SetArrayField("tags", []string{"administrator", "superadmin"}), false},
			},
		},
		{
			name:       "map_array_value_index",
			expression: `http.request.headers["x-categories"][0] == "tech"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetField("http.request.headers", MapValue{"x-categories": ArrayValue{StringValue("tech"), StringValue("news")}}), true},
				{"wrong_value", NewExecutionContext().SetField("http.request.headers", MapValue{"x-categories": ArrayValue{StringValue("sports"), StringValue("news")}}), false},
				{"empty_array", NewExecutionContext().SetField("http.request.headers", MapValue{"x-categories": ArrayValue{}}), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareCustomLists tests custom list references ($list_name) from Cloudflare docs.
func TestCloudflareCustomLists(t *testing.T) {
	schema := NewSchema().
		AddField("ip.src", TypeIP).
		AddField("http.request.uri.path", TypeString).
		AddField("role", TypeString)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "ip_in_custom_list",
			expression: `ip.src in $blocked_ips`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"in_list", NewExecutionContext().SetIPField("ip.src", "10.0.0.5").SetIPList("blocked_ips", []string{"10.0.0.1", "10.0.0.5", "10.0.0.10"}), true},
				{"not_in_list", NewExecutionContext().SetIPField("ip.src", "192.168.1.1").SetIPList("blocked_ips", []string{"10.0.0.1", "10.0.0.5", "10.0.0.10"}), false},
				{"empty_list", NewExecutionContext().SetIPField("ip.src", "10.0.0.5").SetIPList("blocked_ips", []string{}), false},
			},
		},
		{
			name:       "role_in_custom_list",
			expression: `role in $admin_roles`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"in_list", NewExecutionContext().SetStringField("role", "superadmin").SetList("admin_roles", []string{"admin", "superadmin", "moderator"}), true},
				{"not_in_list", NewExecutionContext().SetStringField("role", "user").SetList("admin_roles", []string{"admin", "superadmin", "moderator"}), false},
				{"undefined_list", NewExecutionContext().SetStringField("role", "admin"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareTransformationFunctions tests transformation functions from Cloudflare docs.
func TestCloudflareTransformationFunctions(t *testing.T) {
	schema := NewSchema().
		AddField("http.host", TypeString).
		AddField("http.request.uri.path", TypeString).
		AddField("http.request.uri.query", TypeString).
		AddField("http.request.body.raw", TypeString).
		AddField("tags", TypeArray)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "lower_function",
			expression: `lower(http.host) == "www.cloudflare.com"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"uppercase", NewExecutionContext().SetStringField("http.host", "WWW.CLOUDFLARE.COM"), true},
				{"mixed_case", NewExecutionContext().SetStringField("http.host", "Www.Cloudflare.Com"), true},
				{"already_lower", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com"), true},
				{"different", NewExecutionContext().SetStringField("http.host", "www.example.com"), false},
			},
		},
		{
			name:       "upper_function",
			expression: `upper(http.host) == "WWW.CLOUDFLARE.COM"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"lowercase", NewExecutionContext().SetStringField("http.host", "www.cloudflare.com"), true},
				{"mixed_case", NewExecutionContext().SetStringField("http.host", "Www.Cloudflare.Com"), true},
				{"different", NewExecutionContext().SetStringField("http.host", "www.example.com"), false},
			},
		},
		{
			name:       "len_function",
			expression: `len(http.host) == 11`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"exact", NewExecutionContext().SetStringField("http.host", "example.com"), true},
				{"shorter", NewExecutionContext().SetStringField("http.host", "test.com"), false},
				{"longer", NewExecutionContext().SetStringField("http.host", "www.example.com"), false},
			},
		},
		{
			name:       "substring_function",
			expression: `substring(http.request.body.raw, 0, 5) == "hello"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.request.body.raw", "hello world"), true},
				{"different_prefix", NewExecutionContext().SetStringField("http.request.body.raw", "world hello"), false},
				{"short_string", NewExecutionContext().SetStringField("http.request.body.raw", "hi"), false},
			},
		},
		{
			name:       "url_decode_function",
			expression: `url_decode(http.request.uri.query) contains "John Doe"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"percent_encoded", NewExecutionContext().SetStringField("http.request.uri.query", "name=John%20Doe"), true},
				{"plus_encoded", NewExecutionContext().SetStringField("http.request.uri.query", "name=John+Doe"), true},
				{"different_name", NewExecutionContext().SetStringField("http.request.uri.query", "name=Jane%20Doe"), false},
			},
		},
		{
			name:       "starts_with_function",
			expression: `starts_with(http.request.uri.path, "/blog")`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.request.uri.path", "/blog/articles/2024"), true},
				{"exact", NewExecutionContext().SetStringField("http.request.uri.path", "/blog"), true},
				{"different_prefix", NewExecutionContext().SetStringField("http.request.uri.path", "/api/v1/users"), false},
			},
		},
		{
			name:       "ends_with_function",
			expression: `ends_with(http.request.uri.path, ".html")`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.request.uri.path", "/pages/about.html"), true},
				{"different_suffix", NewExecutionContext().SetStringField("http.request.uri.path", "/pages/about.json"), false},
				{"no_extension", NewExecutionContext().SetStringField("http.request.uri.path", "/api/v1/users"), false},
			},
		},
		{
			name:       "concat_function",
			expression: `concat(http.host, http.request.uri.path) == "example.com/api"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.host", "example.com").SetStringField("http.request.uri.path", "/api"), true},
				{"different_host", NewExecutionContext().SetStringField("http.host", "other.com").SetStringField("http.request.uri.path", "/api"), false},
				{"different_path", NewExecutionContext().SetStringField("http.host", "example.com").SetStringField("http.request.uri.path", "/v1"), false},
			},
		},
		{
			name:       "split_function",
			expression: `split(http.request.uri.path, "/")[1] == "api"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.request.uri.path", "/api/v1/users"), true},
				{"different", NewExecutionContext().SetStringField("http.request.uri.path", "/blog/posts"), false},
			},
		},
		{
			name:       "join_function",
			expression: `join(tags, ",") == "a,b,c"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetArrayField("tags", []string{"a", "b", "c"}), true},
				{"different_order", NewExecutionContext().SetArrayField("tags", []string{"c", "b", "a"}), false},
				{"different_values", NewExecutionContext().SetArrayField("tags", []string{"x", "y", "z"}), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareCollectionFunctions tests array/collection functions from Cloudflare docs.
func TestCloudflareCollectionFunctions(t *testing.T) {
	schema := NewSchema().
		AddField("http.request.headers", TypeMap).
		AddField("http.request.headers.names", TypeArray).
		AddField("tags", TypeArray).
		AddField("values", TypeArray)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "any_function",
			expression: `any(tags[*] == "admin")`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"found", NewExecutionContext().SetArrayField("tags", []string{"user", "admin", "reviewer"}), true},
				{"not_found", NewExecutionContext().SetArrayField("tags", []string{"user", "guest", "reviewer"}), false},
				{"empty_array", NewExecutionContext().SetArrayField("tags", []string{}), false},
			},
		},
		{
			name:       "all_function",
			expression: `all(tags[*] contains "a")`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"all_match", NewExecutionContext().SetArrayField("tags", []string{"alpha", "beta", "gamma"}), true},
				{"some_match", NewExecutionContext().SetArrayField("tags", []string{"alpha", "one", "gamma"}), false},
				{"none_match", NewExecutionContext().SetArrayField("tags", []string{"one", "two", "three"}), false},
			},
		},
		{
			name:       "has_key_function",
			expression: `has_key(http.request.headers, "x-my-header")`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"exists", NewExecutionContext().SetMapField("http.request.headers", map[string]string{"x-my-header": "value"}), true},
				{"not_exists", NewExecutionContext().SetMapField("http.request.headers", map[string]string{"other": "value"}), false},
				{"empty_map", NewExecutionContext().SetMapField("http.request.headers", map[string]string{}), false},
			},
		},
		{
			name:       "has_value_function",
			expression: `has_value(tags, "admin")`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"exists", NewExecutionContext().SetArrayField("tags", []string{"user", "admin", "reviewer"}), true},
				{"not_exists", NewExecutionContext().SetArrayField("tags", []string{"user", "guest", "reviewer"}), false},
				{"empty_array", NewExecutionContext().SetArrayField("tags", []string{}), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareIPFunctions tests IP/network functions from Cloudflare docs.
func TestCloudflareIPFunctions(t *testing.T) {
	schema := NewSchema().
		AddField("ip.src", TypeIP)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "cidr_function_24",
			expression: `cidr(ip.src, 24, 24) in "113.10.0.0/32"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"in_network", NewExecutionContext().SetIPField("ip.src", "113.10.0.2"), true},
				{"in_network_edge", NewExecutionContext().SetIPField("ip.src", "113.10.0.255"), true},
				{"different_network", NewExecutionContext().SetIPField("ip.src", "113.10.1.2"), false},
			},
		},
		{
			name:       "ip_in_cidr",
			expression: `ip.src in "10.0.0.0/8"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"in_range_start", NewExecutionContext().SetIPField("ip.src", "10.0.0.1"), true},
				{"in_range_end", NewExecutionContext().SetIPField("ip.src", "10.255.255.255"), true},
				{"outside_range", NewExecutionContext().SetIPField("ip.src", "192.168.1.1"), false},
			},
		},
		{
			name:       "ipv6_in_cidr",
			expression: `ip.src in "2001:db8::/32"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"in_range", NewExecutionContext().SetIPField("ip.src", "2001:db8:1234::1"), true},
				{"outside_range", NewExecutionContext().SetIPField("ip.src", "2001:db9::1"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareJSONHandling tests JSON-related expressions.
// Note: lookup_json_string and lookup_json_integer functions from Cloudflare docs
// are not implemented in this version. JSON data can be handled by parsing
// it externally and setting fields in the execution context.
func TestCloudflareJSONHandling(t *testing.T) {
	schema := NewSchema().
		AddField("http.request.body.company", TypeString).
		AddField("http.request.body.id", TypeInt)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "json_field_string",
			expression: `http.request.body.company == "cloudflare"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.request.body.company", "cloudflare"), true},
				{"different", NewExecutionContext().SetStringField("http.request.body.company", "google"), false},
				{"empty", NewExecutionContext().SetStringField("http.request.body.company", ""), false},
			},
		},
		{
			name:       "json_field_integer",
			expression: `http.request.body.id == 123`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetIntField("http.request.body.id", 123), true},
				{"different", NewExecutionContext().SetIntField("http.request.body.id", 456), false},
				{"zero", NewExecutionContext().SetIntField("http.request.body.id", 0), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareRealWorldExamples tests real-world expression examples.
func TestCloudflareRealWorldExamples(t *testing.T) {
	schema := NewSchema().
		AddField("http.host", TypeString).
		AddField("http.request.uri.path", TypeString).
		AddField("http.request.uri.query", TypeString).
		AddField("http.request.method", TypeString).
		AddField("http.request.headers", TypeMap).
		AddField("ip.src", TypeIP).
		AddField("ip.src.country", TypeString).
		AddField("cf.bot_management.score", TypeInt).
		AddField("cf.bot_management.verified_bot", TypeBool).
		AddField("ssl", TypeBool)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "geo_block_admin",
			expression: `ip.src.country in {"CN", "RU", "KP"} and http.request.uri.path contains "/admin"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"blocked_country_admin", NewExecutionContext().SetStringField("ip.src.country", "CN").SetStringField("http.request.uri.path", "/admin/dashboard"), true},
				{"blocked_country_not_admin", NewExecutionContext().SetStringField("ip.src.country", "CN").SetStringField("http.request.uri.path", "/api/v1"), false},
				{"allowed_country_admin", NewExecutionContext().SetStringField("ip.src.country", "US").SetStringField("http.request.uri.path", "/admin"), false},
			},
		},
		{
			name:       "bot_protection",
			expression: `(not cf.bot_management.verified_bot) and cf.bot_management.score < 30`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"unverified_low_score", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", false).SetIntField("cf.bot_management.score", 15), true},
				{"unverified_high_score", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", false).SetIntField("cf.bot_management.score", 50), false},
				{"verified_low_score", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", true).SetIntField("cf.bot_management.score", 15), false},
			},
		},
		{
			name:       "require_https_api",
			expression: `http.request.uri.path contains "/api" and (not ssl)`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"api_no_ssl", NewExecutionContext().SetStringField("http.request.uri.path", "/api/v1/users").SetBoolField("ssl", false), true},
				{"api_with_ssl", NewExecutionContext().SetStringField("http.request.uri.path", "/api/v1/users").SetBoolField("ssl", true), false},
				{"non_api_no_ssl", NewExecutionContext().SetStringField("http.request.uri.path", "/pages/about").SetBoolField("ssl", false), false},
			},
		},
		{
			name:       "xss_detection",
			expression: `url_decode(http.request.uri.query) matches "<script|javascript:|onerror="`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"script_tag", NewExecutionContext().SetStringField("http.request.uri.query", "q=%3Cscript%3Ealert(1)%3C/script%3E"), true},
				{"javascript_uri", NewExecutionContext().SetStringField("http.request.uri.query", "url=javascript:alert(1)"), true},
				{"clean_query", NewExecutionContext().SetStringField("http.request.uri.query", "search=hello+world"), false},
			},
		},
		{
			name:       "path_traversal_detection",
			expression: `http.request.uri.path contains ".." or http.request.uri.path contains "%2e%2e"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"double_dot", NewExecutionContext().SetStringField("http.request.uri.path", "/files/../../../etc/passwd"), true},
				{"encoded_dots", NewExecutionContext().SetStringField("http.request.uri.path", "/files/%2e%2e/%2e%2e/etc/passwd"), true},
				{"clean_path", NewExecutionContext().SetStringField("http.request.uri.path", "/files/documents/report.pdf"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareOperatorPrecedence tests operator precedence from Cloudflare docs.
// Reference: NOT (highest) > AND > XOR > OR (lowest)
func TestCloudflareOperatorPrecedence(t *testing.T) {
	schema := NewSchema().
		AddField("a", TypeBool).
		AddField("b", TypeBool).
		AddField("c", TypeBool).
		AddField("d", TypeBool)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "and_before_or",
			expression: `a or b and c`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"a_true", NewExecutionContext().SetBoolField("a", true).SetBoolField("b", false).SetBoolField("c", false), true},
				{"bc_true", NewExecutionContext().SetBoolField("a", false).SetBoolField("b", true).SetBoolField("c", true), true},
				{"b_only_true", NewExecutionContext().SetBoolField("a", false).SetBoolField("b", true).SetBoolField("c", false), false},
			},
		},
		{
			name:       "parentheses_override",
			expression: `(a or b) and c`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"a_c_true", NewExecutionContext().SetBoolField("a", true).SetBoolField("b", false).SetBoolField("c", true), true},
				{"c_false", NewExecutionContext().SetBoolField("a", true).SetBoolField("b", false).SetBoolField("c", false), false},
				{"all_false", NewExecutionContext().SetBoolField("a", false).SetBoolField("b", false).SetBoolField("c", true), false},
			},
		},
		{
			name:       "xor_operator",
			expression: `a xor b`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"a_only", NewExecutionContext().SetBoolField("a", true).SetBoolField("b", false), true},
				{"b_only", NewExecutionContext().SetBoolField("a", false).SetBoolField("b", true), true},
				{"both_true", NewExecutionContext().SetBoolField("a", true).SetBoolField("b", true), false},
				{"both_false", NewExecutionContext().SetBoolField("a", false).SetBoolField("b", false), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareFieldToFieldComparison tests comparing two fields against each other.
func TestCloudflareFieldToFieldComparison(t *testing.T) {
	schema := NewSchema().
		AddField("http.request.headers.origin", TypeString).
		AddField("http.host", TypeString).
		AddField("request.size", TypeInt).
		AddField("response.size", TypeInt)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "string_field_equal",
			expression: `http.request.headers.origin == http.host`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"same", NewExecutionContext().SetStringField("http.request.headers.origin", "example.com").SetStringField("http.host", "example.com"), true},
				{"different", NewExecutionContext().SetStringField("http.request.headers.origin", "other.com").SetStringField("http.host", "example.com"), false},
				{"empty_vs_value", NewExecutionContext().SetStringField("http.request.headers.origin", "").SetStringField("http.host", "example.com"), false},
			},
		},
		{
			name:       "int_field_greater",
			expression: `response.size > request.size`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"greater", NewExecutionContext().SetIntField("response.size", 1000).SetIntField("request.size", 100), true},
				{"equal", NewExecutionContext().SetIntField("response.size", 500).SetIntField("request.size", 500), false},
				{"less", NewExecutionContext().SetIntField("response.size", 100).SetIntField("request.size", 500), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareInlineListFormats tests various inline list formats from Cloudflare docs.
func TestCloudflareInlineListFormats(t *testing.T) {
	schema := NewSchema().
		AddField("http.status", TypeInt).
		AddField("ip.src", TypeIP).
		AddField("http.request.method", TypeString)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "integer_list",
			expression: `http.status in {200, 201, 204}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"in_list", NewExecutionContext().SetIntField("http.status", 201), true},
				{"not_in_list", NewExecutionContext().SetIntField("http.status", 404), false},
				{"first_item", NewExecutionContext().SetIntField("http.status", 200), true},
			},
		},
		{
			name:       "integer_range_list",
			expression: `http.status in {200..299, 400..499}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"in_first_range", NewExecutionContext().SetIntField("http.status", 250), true},
				{"in_second_range", NewExecutionContext().SetIntField("http.status", 404), true},
				{"between_ranges", NewExecutionContext().SetIntField("http.status", 350), false},
			},
		},
		{
			name:       "string_list",
			expression: `http.request.method in {"GET", "POST", "PUT"}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"in_list", NewExecutionContext().SetStringField("http.request.method", "PUT"), true},
				{"not_in_list", NewExecutionContext().SetStringField("http.request.method", "DELETE"), false},
				{"case_sensitive", NewExecutionContext().SetStringField("http.request.method", "get"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareEdgeCases tests edge cases and boundary conditions.
func TestCloudflareEdgeCases(t *testing.T) {
	schema := NewSchema().
		AddField("http.host", TypeString).
		AddField("http.request.uri.path", TypeString).
		AddField("tags", TypeArray).
		AddField("data", TypeMap)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "empty_string_equality",
			expression: `http.host == ""`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"empty_match", NewExecutionContext().SetStringField("http.host", ""), true},
				{"non_empty", NewExecutionContext().SetStringField("http.host", "example.com"), false},
			},
		},
		{
			name:       "array_unpack",
			expression: `tags[*] == "test"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"found", NewExecutionContext().SetArrayField("tags", []string{"a", "test", "b"}), true},
				{"not_found", NewExecutionContext().SetArrayField("tags", []string{"a", "b"}), false},
				{"empty_array", NewExecutionContext().SetArrayField("tags", []string{}), false},
			},
		},
		{
			name:       "array_out_of_bounds",
			expression: `tags[10] == "test"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"out_of_bounds", NewExecutionContext().SetArrayField("tags", []string{"a", "b"}), false},
				{"empty_array", NewExecutionContext().SetArrayField("tags", []string{}), false},
			},
		},
		{
			name:       "map_key_access",
			expression: `data["key"] == "value"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"exists", NewExecutionContext().SetMapField("data", map[string]string{"key": "value"}), true},
				{"wrong_value", NewExecutionContext().SetMapField("data", map[string]string{"key": "other"}), false},
				{"missing_key", NewExecutionContext().SetMapField("data", map[string]string{"other": "value"}), false},
			},
		},
		{
			name:       "chained_function",
			expression: `len(lower(http.host)) == 11`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"uppercase", NewExecutionContext().SetStringField("http.host", "EXAMPLE.COM"), true},
				{"lowercase", NewExecutionContext().SetStringField("http.host", "example.com"), true},
				{"different_length", NewExecutionContext().SetStringField("http.host", "test.com"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareWAFFields tests WAF-specific fields from Cloudflare docs.
// Reference: https://developers.cloudflare.com/ruleset-engine/rules-language/fields/dynamic-fields/
func TestCloudflareWAFFields(t *testing.T) {
	schema := NewSchema().
		// WAF scoring fields
		AddField("cf.waf.score", TypeInt).
		AddField("cf.waf.score.sqli", TypeInt).
		AddField("cf.waf.score.xss", TypeInt).
		AddField("cf.waf.score.rce", TypeInt).
		AddField("cf.waf.score.class", TypeString).
		AddField("cf.threat_score", TypeInt).
		// WAF credential check fields
		AddField("cf.waf.auth_detected", TypeBool).
		AddField("cf.waf.credential_check.password_leaked", TypeBool).
		AddField("cf.waf.credential_check.username_leaked", TypeBool).
		AddField("cf.waf.credential_check.username_and_password_leaked", TypeBool).
		// WAF content scan fields
		AddField("cf.waf.content_scan.has_obj", TypeBool).
		AddField("cf.waf.content_scan.has_malicious_obj", TypeBool).
		AddField("cf.waf.content_scan.num_obj", TypeInt).
		AddField("cf.waf.content_scan.num_malicious_obj", TypeInt).
		AddField("cf.waf.content_scan.obj_types", TypeArray).
		AddField("cf.waf.content_scan.obj_sizes", TypeArray).
		// Request fields for context
		AddField("http.request.uri.path", TypeString).
		AddField("http.request.method", TypeString).
		AddField("ip.src", TypeIP)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "waf_score_high_risk",
			expression: `cf.waf.score < 20`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"score_15", NewExecutionContext().SetIntField("cf.waf.score", 15), true},
				{"score_5", NewExecutionContext().SetIntField("cf.waf.score", 5), true},
				{"score_20", NewExecutionContext().SetIntField("cf.waf.score", 20), false},
				{"score_50", NewExecutionContext().SetIntField("cf.waf.score", 50), false},
			},
		},
		{
			name:       "waf_score_low_risk",
			expression: `cf.waf.score >= 80`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"score_85", NewExecutionContext().SetIntField("cf.waf.score", 85), true},
				{"score_80", NewExecutionContext().SetIntField("cf.waf.score", 80), true},
				{"score_79", NewExecutionContext().SetIntField("cf.waf.score", 79), false},
				{"score_50", NewExecutionContext().SetIntField("cf.waf.score", 50), false},
			},
		},
		{
			name:       "waf_score_medium_risk_range",
			expression: `cf.waf.score >= 20 and cf.waf.score < 80`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"score_50", NewExecutionContext().SetIntField("cf.waf.score", 50), true},
				{"score_20", NewExecutionContext().SetIntField("cf.waf.score", 20), true},
				{"score_79", NewExecutionContext().SetIntField("cf.waf.score", 79), true},
				{"score_19", NewExecutionContext().SetIntField("cf.waf.score", 19), false},
				{"score_80", NewExecutionContext().SetIntField("cf.waf.score", 80), false},
			},
		},
		{
			name:       "waf_sqli_score_high_risk",
			expression: `cf.waf.score.sqli < 30`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"score_10", NewExecutionContext().SetIntField("cf.waf.score.sqli", 10), true},
				{"score_29", NewExecutionContext().SetIntField("cf.waf.score.sqli", 29), true},
				{"score_30", NewExecutionContext().SetIntField("cf.waf.score.sqli", 30), false},
				{"score_80", NewExecutionContext().SetIntField("cf.waf.score.sqli", 80), false},
			},
		},
		{
			name:       "waf_sqli_with_path_check",
			expression: `cf.waf.score.sqli < 50 and http.request.uri.path contains "/api"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"low_score_api_path", NewExecutionContext().SetIntField("cf.waf.score.sqli", 25).SetStringField("http.request.uri.path", "/api/v1/users"), true},
				{"low_score_non_api", NewExecutionContext().SetIntField("cf.waf.score.sqli", 25).SetStringField("http.request.uri.path", "/home"), false},
				{"high_score_api_path", NewExecutionContext().SetIntField("cf.waf.score.sqli", 60).SetStringField("http.request.uri.path", "/api/v1/users"), false},
			},
		},
		{
			name:       "waf_xss_score_high_risk",
			expression: `cf.waf.score.xss < 30`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"score_15", NewExecutionContext().SetIntField("cf.waf.score.xss", 15), true},
				{"score_30", NewExecutionContext().SetIntField("cf.waf.score.xss", 30), false},
				{"score_80", NewExecutionContext().SetIntField("cf.waf.score.xss", 80), false},
			},
		},
		{
			name:       "waf_rce_score_critical",
			expression: `cf.waf.score.rce < 20`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"score_5", NewExecutionContext().SetIntField("cf.waf.score.rce", 5), true},
				{"score_19", NewExecutionContext().SetIntField("cf.waf.score.rce", 19), true},
				{"score_20", NewExecutionContext().SetIntField("cf.waf.score.rce", 20), false},
				{"score_50", NewExecutionContext().SetIntField("cf.waf.score.rce", 50), false},
			},
		},
		{
			name:       "waf_score_class_attack",
			expression: `cf.waf.score.class == "attack"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"attack", NewExecutionContext().SetStringField("cf.waf.score.class", "attack"), true},
				{"likely_attack", NewExecutionContext().SetStringField("cf.waf.score.class", "likely_attack"), false},
				{"benign", NewExecutionContext().SetStringField("cf.waf.score.class", "benign"), false},
			},
		},
		{
			name:       "waf_score_class_likely_attack",
			expression: `cf.waf.score.class in {"attack", "likely_attack"}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"attack", NewExecutionContext().SetStringField("cf.waf.score.class", "attack"), true},
				{"likely_attack", NewExecutionContext().SetStringField("cf.waf.score.class", "likely_attack"), true},
				{"benign", NewExecutionContext().SetStringField("cf.waf.score.class", "benign"), false},
				{"unknown", NewExecutionContext().SetStringField("cf.waf.score.class", "unknown"), false},
			},
		},
		{
			name:       "threat_score_high",
			expression: `cf.threat_score > 50`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"score_75", NewExecutionContext().SetIntField("cf.threat_score", 75), true},
				{"score_51", NewExecutionContext().SetIntField("cf.threat_score", 51), true},
				{"score_50", NewExecutionContext().SetIntField("cf.threat_score", 50), false},
				{"score_25", NewExecutionContext().SetIntField("cf.threat_score", 25), false},
			},
		},
		{
			name:       "combined_waf_scores_any_high_risk",
			expression: `cf.waf.score.sqli < 30 or cf.waf.score.xss < 30 or cf.waf.score.rce < 30`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"xss_high_risk", NewExecutionContext().SetIntField("cf.waf.score.sqli", 80).SetIntField("cf.waf.score.xss", 20).SetIntField("cf.waf.score.rce", 90), true},
				{"sqli_high_risk", NewExecutionContext().SetIntField("cf.waf.score.sqli", 15).SetIntField("cf.waf.score.xss", 80).SetIntField("cf.waf.score.rce", 90), true},
				{"rce_high_risk", NewExecutionContext().SetIntField("cf.waf.score.sqli", 80).SetIntField("cf.waf.score.xss", 80).SetIntField("cf.waf.score.rce", 10), true},
				{"all_low_risk", NewExecutionContext().SetIntField("cf.waf.score.sqli", 80).SetIntField("cf.waf.score.xss", 80).SetIntField("cf.waf.score.rce", 80), false},
			},
		},
		{
			name:       "auth_detected_block_non_post",
			expression: `cf.waf.auth_detected and http.request.method != "POST"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"auth_get", NewExecutionContext().SetBoolField("cf.waf.auth_detected", true).SetStringField("http.request.method", "GET"), true},
				{"auth_put", NewExecutionContext().SetBoolField("cf.waf.auth_detected", true).SetStringField("http.request.method", "PUT"), true},
				{"auth_post", NewExecutionContext().SetBoolField("cf.waf.auth_detected", true).SetStringField("http.request.method", "POST"), false},
				{"no_auth_get", NewExecutionContext().SetBoolField("cf.waf.auth_detected", false).SetStringField("http.request.method", "GET"), false},
			},
		},
		{
			name:       "password_leaked",
			expression: `cf.waf.credential_check.password_leaked`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"leaked", NewExecutionContext().SetBoolField("cf.waf.credential_check.password_leaked", true), true},
				{"not_leaked", NewExecutionContext().SetBoolField("cf.waf.credential_check.password_leaked", false), false},
			},
		},
		{
			name:       "any_credential_leak",
			expression: `cf.waf.credential_check.password_leaked or cf.waf.credential_check.username_leaked or cf.waf.credential_check.username_and_password_leaked`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"username_leaked", NewExecutionContext().SetBoolField("cf.waf.credential_check.password_leaked", false).SetBoolField("cf.waf.credential_check.username_leaked", true).SetBoolField("cf.waf.credential_check.username_and_password_leaked", false), true},
				{"password_leaked", NewExecutionContext().SetBoolField("cf.waf.credential_check.password_leaked", true).SetBoolField("cf.waf.credential_check.username_leaked", false).SetBoolField("cf.waf.credential_check.username_and_password_leaked", false), true},
				{"both_leaked", NewExecutionContext().SetBoolField("cf.waf.credential_check.password_leaked", false).SetBoolField("cf.waf.credential_check.username_leaked", false).SetBoolField("cf.waf.credential_check.username_and_password_leaked", true), true},
				{"none_leaked", NewExecutionContext().SetBoolField("cf.waf.credential_check.password_leaked", false).SetBoolField("cf.waf.credential_check.username_leaked", false).SetBoolField("cf.waf.credential_check.username_and_password_leaked", false), false},
			},
		},
		{
			name:       "has_malicious_content",
			expression: `cf.waf.content_scan.has_malicious_obj`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"has_malicious", NewExecutionContext().SetBoolField("cf.waf.content_scan.has_malicious_obj", true), true},
				{"no_malicious", NewExecutionContext().SetBoolField("cf.waf.content_scan.has_malicious_obj", false), false},
			},
		},
		{
			name:       "multiple_malicious_objects",
			expression: `cf.waf.content_scan.num_malicious_obj > 1`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"count_3", NewExecutionContext().SetIntField("cf.waf.content_scan.num_malicious_obj", 3), true},
				{"count_2", NewExecutionContext().SetIntField("cf.waf.content_scan.num_malicious_obj", 2), true},
				{"count_1", NewExecutionContext().SetIntField("cf.waf.content_scan.num_malicious_obj", 1), false},
				{"count_0", NewExecutionContext().SetIntField("cf.waf.content_scan.num_malicious_obj", 0), false},
			},
		},
		{
			name:       "content_scan_with_file_check",
			expression: `cf.waf.content_scan.has_obj and cf.waf.content_scan.num_obj > 5`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"has_many_obj", NewExecutionContext().SetBoolField("cf.waf.content_scan.has_obj", true).SetIntField("cf.waf.content_scan.num_obj", 10), true},
				{"has_few_obj", NewExecutionContext().SetBoolField("cf.waf.content_scan.has_obj", true).SetIntField("cf.waf.content_scan.num_obj", 3), false},
				{"no_obj_many", NewExecutionContext().SetBoolField("cf.waf.content_scan.has_obj", false).SetIntField("cf.waf.content_scan.num_obj", 10), false},
			},
		},
		{
			name:       "suspicious_file_type",
			expression: `cf.waf.content_scan.obj_types[*] == "application/x-executable"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"has_executable", NewExecutionContext().SetArrayField("cf.waf.content_scan.obj_types", []string{"image/png", "application/x-executable"}), true},
				{"only_images", NewExecutionContext().SetArrayField("cf.waf.content_scan.obj_types", []string{"image/png", "image/jpeg"}), false},
				{"empty_types", NewExecutionContext().SetArrayField("cf.waf.content_scan.obj_types", []string{}), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareBotManagement tests bot management fields from Cloudflare docs.
func TestCloudflareBotManagement(t *testing.T) {
	schema := NewSchema().
		AddField("cf.bot_management.score", TypeInt).
		AddField("cf.bot_management.verified_bot", TypeBool).
		AddField("cf.bot_management.js_detection.passed", TypeBool).
		AddField("cf.bot_management.ja3_hash", TypeString).
		AddField("cf.bot_management.ja4", TypeString).
		AddField("cf.bot_management.detection_ids", TypeArray).
		AddField("cf.bot_management.corporate_proxy", TypeBool).
		AddField("cf.bot_management.static_resource", TypeBool).
		AddField("http.request.uri.path", TypeString).
		AddField("http.user_agent", TypeString)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "bot_score_likely_bot",
			expression: `cf.bot_management.score < 30`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"score_15", NewExecutionContext().SetIntField("cf.bot_management.score", 15), true},
				{"score_29", NewExecutionContext().SetIntField("cf.bot_management.score", 29), true},
				{"score_30", NewExecutionContext().SetIntField("cf.bot_management.score", 30), false},
				{"score_70", NewExecutionContext().SetIntField("cf.bot_management.score", 70), false},
			},
		},
		{
			name:       "bot_score_likely_human",
			expression: `cf.bot_management.score >= 70`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"score_85", NewExecutionContext().SetIntField("cf.bot_management.score", 85), true},
				{"score_70", NewExecutionContext().SetIntField("cf.bot_management.score", 70), true},
				{"score_69", NewExecutionContext().SetIntField("cf.bot_management.score", 69), false},
				{"score_30", NewExecutionContext().SetIntField("cf.bot_management.score", 30), false},
			},
		},
		{
			name:       "verified_bot_allowed",
			expression: `cf.bot_management.verified_bot`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"verified", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", true), true},
				{"not_verified", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", false), false},
			},
		},
		{
			name:       "block_unverified_low_score",
			expression: `(not cf.bot_management.verified_bot) and cf.bot_management.score < 30`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"unverified_low_score", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", false).SetIntField("cf.bot_management.score", 20), true},
				{"verified_low_score", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", true).SetIntField("cf.bot_management.score", 20), false},
				{"unverified_high_score", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", false).SetIntField("cf.bot_management.score", 50), false},
			},
		},
		{
			name:       "js_detection_passed",
			expression: `cf.bot_management.js_detection.passed`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"passed", NewExecutionContext().SetBoolField("cf.bot_management.js_detection.passed", true), true},
				{"failed", NewExecutionContext().SetBoolField("cf.bot_management.js_detection.passed", false), false},
			},
		},
		{
			name:       "js_detection_failed_low_score",
			expression: `(not cf.bot_management.js_detection.passed) and cf.bot_management.score < 50`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"failed_low_score", NewExecutionContext().SetBoolField("cf.bot_management.js_detection.passed", false).SetIntField("cf.bot_management.score", 30), true},
				{"passed_low_score", NewExecutionContext().SetBoolField("cf.bot_management.js_detection.passed", true).SetIntField("cf.bot_management.score", 30), false},
				{"failed_high_score", NewExecutionContext().SetBoolField("cf.bot_management.js_detection.passed", false).SetIntField("cf.bot_management.score", 60), false},
			},
		},
		{
			name:       "ja3_hash_match",
			expression: `cf.bot_management.ja3_hash == "e7d705a3286e19ea42f587b344ee6865"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("cf.bot_management.ja3_hash", "e7d705a3286e19ea42f587b344ee6865"), true},
				{"no_match", NewExecutionContext().SetStringField("cf.bot_management.ja3_hash", "different_hash"), false},
				{"empty", NewExecutionContext().SetStringField("cf.bot_management.ja3_hash", ""), false},
			},
		},
		{
			name:       "ja3_hash_blocklist",
			expression: `cf.bot_management.ja3_hash in $blocked_ja3`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"in_list", NewExecutionContext().SetStringField("cf.bot_management.ja3_hash", "malicious_fingerprint").SetList("blocked_ja3", []string{"bad_fingerprint1", "malicious_fingerprint", "bad_fingerprint2"}), true},
				{"not_in_list", NewExecutionContext().SetStringField("cf.bot_management.ja3_hash", "good_fingerprint").SetList("blocked_ja3", []string{"bad_fingerprint1", "malicious_fingerprint", "bad_fingerprint2"}), false},
			},
		},
		{
			name:       "ja4_fingerprint_check",
			expression: `cf.bot_management.ja4 contains "t13d"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"contains_t13d", NewExecutionContext().SetStringField("cf.bot_management.ja4", "t13d1516h2_8daaf6152771_b0da82dd1658"), true},
				{"no_t13d", NewExecutionContext().SetStringField("cf.bot_management.ja4", "t12d1516h2_8daaf6152771_b0da82dd1658"), false},
				{"empty", NewExecutionContext().SetStringField("cf.bot_management.ja4", ""), false},
			},
		},
		{
			name:       "detection_id_match",
			expression: `cf.bot_management.detection_ids[*] == "automation"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"has_automation", NewExecutionContext().SetArrayField("cf.bot_management.detection_ids", []string{"headless_browser", "automation"}), true},
				{"no_automation", NewExecutionContext().SetArrayField("cf.bot_management.detection_ids", []string{"headless_browser", "suspicious_ua"}), false},
				{"empty_array", NewExecutionContext().SetArrayField("cf.bot_management.detection_ids", []string{}), false},
			},
		},
		{
			name:       "any_detection_id",
			expression: `any(cf.bot_management.detection_ids[*] == "headless_browser")`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"has_headless", NewExecutionContext().SetArrayField("cf.bot_management.detection_ids", []string{"headless_browser", "suspicious_ua"}), true},
				{"no_headless", NewExecutionContext().SetArrayField("cf.bot_management.detection_ids", []string{"automation", "suspicious_ua"}), false},
			},
		},
		{
			name:       "corporate_proxy_allowed",
			expression: `cf.bot_management.corporate_proxy`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"is_proxy", NewExecutionContext().SetBoolField("cf.bot_management.corporate_proxy", true), true},
				{"not_proxy", NewExecutionContext().SetBoolField("cf.bot_management.corporate_proxy", false), false},
			},
		},
		{
			name:       "static_resource_skip",
			expression: `cf.bot_management.static_resource`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"is_static", NewExecutionContext().SetBoolField("cf.bot_management.static_resource", true), true},
				{"not_static", NewExecutionContext().SetBoolField("cf.bot_management.static_resource", false), false},
			},
		},
		{
			name:       "complex_bot_protection",
			expression: `(not cf.bot_management.verified_bot) and cf.bot_management.score < 30 and (not cf.bot_management.static_resource)`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"unverified_low_score_dynamic", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", false).SetIntField("cf.bot_management.score", 15).SetBoolField("cf.bot_management.static_resource", false), true},
				{"verified_low_score_dynamic", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", true).SetIntField("cf.bot_management.score", 15).SetBoolField("cf.bot_management.static_resource", false), false},
				{"unverified_high_score_dynamic", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", false).SetIntField("cf.bot_management.score", 50).SetBoolField("cf.bot_management.static_resource", false), false},
				{"unverified_low_score_static", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", false).SetIntField("cf.bot_management.score", 15).SetBoolField("cf.bot_management.static_resource", true), false},
			},
		},
		{
			name:       "api_bot_protection",
			expression: `http.request.uri.path contains "/api" and (not cf.bot_management.verified_bot) and cf.bot_management.score < 50`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"api_unverified_low_score", NewExecutionContext().SetStringField("http.request.uri.path", "/api/v1/data").SetBoolField("cf.bot_management.verified_bot", false).SetIntField("cf.bot_management.score", 25), true},
				{"non_api_unverified_low_score", NewExecutionContext().SetStringField("http.request.uri.path", "/home").SetBoolField("cf.bot_management.verified_bot", false).SetIntField("cf.bot_management.score", 25), false},
				{"api_verified_low_score", NewExecutionContext().SetStringField("http.request.uri.path", "/api/v1/data").SetBoolField("cf.bot_management.verified_bot", true).SetIntField("cf.bot_management.score", 25), false},
				{"api_unverified_high_score", NewExecutionContext().SetStringField("http.request.uri.path", "/api/v1/data").SetBoolField("cf.bot_management.verified_bot", false).SetIntField("cf.bot_management.score", 60), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareLLMSecurity tests LLM security fields from Cloudflare docs.
func TestCloudflareLLMSecurity(t *testing.T) {
	schema := NewSchema().
		AddField("cf.llm.prompt.detected", TypeBool).
		AddField("cf.llm.prompt.injection_score", TypeInt).
		AddField("cf.llm.prompt.pii_detected", TypeBool).
		AddField("cf.llm.prompt.pii_categories", TypeArray).
		AddField("cf.llm.prompt.unsafe_topic_detected", TypeBool).
		AddField("cf.llm.prompt.unsafe_topic_categories", TypeArray).
		AddField("http.request.uri.path", TypeString)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "llm_prompt_detected",
			expression: `cf.llm.prompt.detected`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"detected", NewExecutionContext().SetBoolField("cf.llm.prompt.detected", true), true},
				{"not_detected", NewExecutionContext().SetBoolField("cf.llm.prompt.detected", false), false},
			},
		},
		{
			name:       "prompt_injection_high_risk",
			expression: `cf.llm.prompt.injection_score < 30`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"score_15", NewExecutionContext().SetIntField("cf.llm.prompt.injection_score", 15), true},
				{"score_29", NewExecutionContext().SetIntField("cf.llm.prompt.injection_score", 29), true},
				{"score_30", NewExecutionContext().SetIntField("cf.llm.prompt.injection_score", 30), false},
				{"score_80", NewExecutionContext().SetIntField("cf.llm.prompt.injection_score", 80), false},
			},
		},
		{
			name:       "prompt_injection_with_prompt_detected",
			expression: `cf.llm.prompt.detected and cf.llm.prompt.injection_score < 50`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"detected_low_score", NewExecutionContext().SetBoolField("cf.llm.prompt.detected", true).SetIntField("cf.llm.prompt.injection_score", 25), true},
				{"not_detected_low_score", NewExecutionContext().SetBoolField("cf.llm.prompt.detected", false).SetIntField("cf.llm.prompt.injection_score", 25), false},
				{"detected_high_score", NewExecutionContext().SetBoolField("cf.llm.prompt.detected", true).SetIntField("cf.llm.prompt.injection_score", 60), false},
			},
		},
		{
			name:       "pii_detected",
			expression: `cf.llm.prompt.pii_detected`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"pii_found", NewExecutionContext().SetBoolField("cf.llm.prompt.pii_detected", true), true},
				{"no_pii", NewExecutionContext().SetBoolField("cf.llm.prompt.pii_detected", false), false},
			},
		},
		{
			name:       "pii_category_ssn",
			expression: `cf.llm.prompt.pii_categories[*] == "ssn"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"has_ssn", NewExecutionContext().SetArrayField("cf.llm.prompt.pii_categories", []string{"email", "ssn", "phone"}), true},
				{"no_ssn", NewExecutionContext().SetArrayField("cf.llm.prompt.pii_categories", []string{"email", "phone"}), false},
				{"empty_categories", NewExecutionContext().SetArrayField("cf.llm.prompt.pii_categories", []string{}), false},
			},
		},
		{
			name:       "any_sensitive_pii",
			expression: `any(cf.llm.prompt.pii_categories[*] == "ssn") or any(cf.llm.prompt.pii_categories[*] == "credit_card")`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"has_credit_card", NewExecutionContext().SetArrayField("cf.llm.prompt.pii_categories", []string{"email", "credit_card"}), true},
				{"has_ssn", NewExecutionContext().SetArrayField("cf.llm.prompt.pii_categories", []string{"ssn", "phone"}), true},
				{"has_both", NewExecutionContext().SetArrayField("cf.llm.prompt.pii_categories", []string{"ssn", "credit_card"}), true},
				{"neither", NewExecutionContext().SetArrayField("cf.llm.prompt.pii_categories", []string{"email", "phone"}), false},
			},
		},
		{
			name:       "unsafe_topic_detected",
			expression: `cf.llm.prompt.unsafe_topic_detected`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"unsafe", NewExecutionContext().SetBoolField("cf.llm.prompt.unsafe_topic_detected", true), true},
				{"safe", NewExecutionContext().SetBoolField("cf.llm.prompt.unsafe_topic_detected", false), false},
			},
		},
		{
			name:       "unsafe_topic_violence",
			expression: `cf.llm.prompt.unsafe_topic_categories[*] == "violence"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"has_violence", NewExecutionContext().SetArrayField("cf.llm.prompt.unsafe_topic_categories", []string{"violence", "hate_speech"}), true},
				{"no_violence", NewExecutionContext().SetArrayField("cf.llm.prompt.unsafe_topic_categories", []string{"hate_speech", "illegal_content"}), false},
				{"empty", NewExecutionContext().SetArrayField("cf.llm.prompt.unsafe_topic_categories", []string{}), false},
			},
		},
		{
			name:       "llm_endpoint_protection",
			expression: `http.request.uri.path contains "/api/chat" and (cf.llm.prompt.injection_score < 30 or cf.llm.prompt.unsafe_topic_detected)`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"chat_unsafe", NewExecutionContext().SetStringField("http.request.uri.path", "/api/chat/completions").SetIntField("cf.llm.prompt.injection_score", 50).SetBoolField("cf.llm.prompt.unsafe_topic_detected", true), true},
				{"chat_injection", NewExecutionContext().SetStringField("http.request.uri.path", "/api/chat/completions").SetIntField("cf.llm.prompt.injection_score", 20).SetBoolField("cf.llm.prompt.unsafe_topic_detected", false), true},
				{"chat_safe", NewExecutionContext().SetStringField("http.request.uri.path", "/api/chat/completions").SetIntField("cf.llm.prompt.injection_score", 50).SetBoolField("cf.llm.prompt.unsafe_topic_detected", false), false},
				{"non_chat_unsafe", NewExecutionContext().SetStringField("http.request.uri.path", "/api/users").SetIntField("cf.llm.prompt.injection_score", 50).SetBoolField("cf.llm.prompt.unsafe_topic_detected", true), false},
			},
		},
		{
			name:       "block_pii_in_prompts",
			expression: `cf.llm.prompt.detected and cf.llm.prompt.pii_detected`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"prompt_with_pii", NewExecutionContext().SetBoolField("cf.llm.prompt.detected", true).SetBoolField("cf.llm.prompt.pii_detected", true), true},
				{"prompt_no_pii", NewExecutionContext().SetBoolField("cf.llm.prompt.detected", true).SetBoolField("cf.llm.prompt.pii_detected", false), false},
				{"no_prompt_with_pii", NewExecutionContext().SetBoolField("cf.llm.prompt.detected", false).SetBoolField("cf.llm.prompt.pii_detected", true), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareWAFRealWorldRules tests real-world WAF rule examples.
func TestCloudflareWAFRealWorldRules(t *testing.T) {
	schema := NewSchema().
		// WAF fields
		AddField("cf.waf.score", TypeInt).
		AddField("cf.waf.score.sqli", TypeInt).
		AddField("cf.waf.score.xss", TypeInt).
		AddField("cf.waf.score.rce", TypeInt).
		AddField("cf.threat_score", TypeInt).
		AddField("cf.waf.credential_check.password_leaked", TypeBool).
		AddField("cf.waf.content_scan.has_malicious_obj", TypeBool).
		// Bot management
		AddField("cf.bot_management.score", TypeInt).
		AddField("cf.bot_management.verified_bot", TypeBool).
		AddField("cf.bot_management.ja3_hash", TypeString).
		// HTTP fields
		AddField("http.request.uri.path", TypeString).
		AddField("http.request.uri.query", TypeString).
		AddField("http.request.method", TypeString).
		AddField("http.request.headers", TypeMap).
		AddField("http.host", TypeString).
		// IP fields
		AddField("ip.src", TypeIP).
		AddField("ip.src.country", TypeString)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		{
			name:       "block_high_risk_waf",
			expression: `cf.waf.score < 20 and (not cf.bot_management.verified_bot)`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"low_score_unverified", NewExecutionContext().SetIntField("cf.waf.score", 10).SetBoolField("cf.bot_management.verified_bot", false), true},
				{"low_score_verified", NewExecutionContext().SetIntField("cf.waf.score", 10).SetBoolField("cf.bot_management.verified_bot", true), false},
				{"high_score_unverified", NewExecutionContext().SetIntField("cf.waf.score", 50).SetBoolField("cf.bot_management.verified_bot", false), false},
			},
		},
		{
			name:       "sqli_protection_login",
			expression: `http.request.uri.path contains "/login" and cf.waf.score.sqli < 40`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"login_low_sqli", NewExecutionContext().SetStringField("http.request.uri.path", "/api/login").SetIntField("cf.waf.score.sqli", 25), true},
				{"login_high_sqli", NewExecutionContext().SetStringField("http.request.uri.path", "/api/login").SetIntField("cf.waf.score.sqli", 50), false},
				{"non_login_low_sqli", NewExecutionContext().SetStringField("http.request.uri.path", "/api/users").SetIntField("cf.waf.score.sqli", 25), false},
			},
		},
		{
			name:       "xss_protection_comments",
			expression: `(http.request.uri.path contains "/comment" or http.request.uri.path contains "/post") and cf.waf.score.xss < 50`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"comment_low_xss", NewExecutionContext().SetStringField("http.request.uri.path", "/api/comment/new").SetIntField("cf.waf.score.xss", 30), true},
				{"post_low_xss", NewExecutionContext().SetStringField("http.request.uri.path", "/api/post/create").SetIntField("cf.waf.score.xss", 30), true},
				{"comment_high_xss", NewExecutionContext().SetStringField("http.request.uri.path", "/api/comment/new").SetIntField("cf.waf.score.xss", 60), false},
				{"other_path_low_xss", NewExecutionContext().SetStringField("http.request.uri.path", "/api/users").SetIntField("cf.waf.score.xss", 30), false},
			},
		},
		{
			name:       "rce_protection_admin",
			expression: `http.request.uri.path contains "/admin" and cf.waf.score.rce < 30`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"admin_low_rce", NewExecutionContext().SetStringField("http.request.uri.path", "/admin/execute").SetIntField("cf.waf.score.rce", 15), true},
				{"admin_high_rce", NewExecutionContext().SetStringField("http.request.uri.path", "/admin/execute").SetIntField("cf.waf.score.rce", 50), false},
				{"non_admin_low_rce", NewExecutionContext().SetStringField("http.request.uri.path", "/api/users").SetIntField("cf.waf.score.rce", 15), false},
			},
		},
		{
			name:       "credential_stuffing_protection",
			expression: `http.request.uri.path contains "/login" and http.request.method == "POST" and cf.waf.credential_check.password_leaked`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"login_post_leaked", NewExecutionContext().SetStringField("http.request.uri.path", "/api/login").SetStringField("http.request.method", "POST").SetBoolField("cf.waf.credential_check.password_leaked", true), true},
				{"login_post_not_leaked", NewExecutionContext().SetStringField("http.request.uri.path", "/api/login").SetStringField("http.request.method", "POST").SetBoolField("cf.waf.credential_check.password_leaked", false), false},
				{"login_get_leaked", NewExecutionContext().SetStringField("http.request.uri.path", "/api/login").SetStringField("http.request.method", "GET").SetBoolField("cf.waf.credential_check.password_leaked", true), false},
			},
		},
		{
			name:       "malware_upload_protection",
			expression: `http.request.uri.path contains "/upload" and cf.waf.content_scan.has_malicious_obj`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"upload_malicious", NewExecutionContext().SetStringField("http.request.uri.path", "/api/upload/file").SetBoolField("cf.waf.content_scan.has_malicious_obj", true), true},
				{"upload_clean", NewExecutionContext().SetStringField("http.request.uri.path", "/api/upload/file").SetBoolField("cf.waf.content_scan.has_malicious_obj", false), false},
				{"non_upload_malicious", NewExecutionContext().SetStringField("http.request.uri.path", "/api/download").SetBoolField("cf.waf.content_scan.has_malicious_obj", true), false},
			},
		},
		{
			name:       "api_bot_protection_strict",
			expression: `http.request.uri.path contains "/api" and cf.bot_management.score < 30 and (not cf.bot_management.verified_bot)`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"api_low_score_unverified", NewExecutionContext().SetStringField("http.request.uri.path", "/api/v2/data").SetIntField("cf.bot_management.score", 20).SetBoolField("cf.bot_management.verified_bot", false), true},
				{"api_low_score_verified", NewExecutionContext().SetStringField("http.request.uri.path", "/api/v2/data").SetIntField("cf.bot_management.score", 20).SetBoolField("cf.bot_management.verified_bot", true), false},
				{"api_high_score_unverified", NewExecutionContext().SetStringField("http.request.uri.path", "/api/v2/data").SetIntField("cf.bot_management.score", 50).SetBoolField("cf.bot_management.verified_bot", false), false},
				{"non_api_low_score", NewExecutionContext().SetStringField("http.request.uri.path", "/home").SetIntField("cf.bot_management.score", 20).SetBoolField("cf.bot_management.verified_bot", false), false},
			},
		},
		{
			name:       "geo_threat_protection",
			expression: `ip.src.country in {"CN", "RU", "KP"} and cf.threat_score > 30`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"ru_high_threat", NewExecutionContext().SetStringField("ip.src.country", "RU").SetIntField("cf.threat_score", 50), true},
				{"cn_high_threat", NewExecutionContext().SetStringField("ip.src.country", "CN").SetIntField("cf.threat_score", 50), true},
				{"ru_low_threat", NewExecutionContext().SetStringField("ip.src.country", "RU").SetIntField("cf.threat_score", 20), false},
				{"us_high_threat", NewExecutionContext().SetStringField("ip.src.country", "US").SetIntField("cf.threat_score", 50), false},
			},
		},
		{
			name:       "ja3_fingerprint_block",
			expression: `cf.bot_management.ja3_hash in $malicious_ja3_hashes`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"in_list", NewExecutionContext().SetStringField("cf.bot_management.ja3_hash", "e7d705a3286e19ea42f587b344ee6865").SetList("malicious_ja3_hashes", []string{"e7d705a3286e19ea42f587b344ee6865", "other_bad_hash"}), true},
				{"not_in_list", NewExecutionContext().SetStringField("cf.bot_management.ja3_hash", "good_hash").SetList("malicious_ja3_hashes", []string{"e7d705a3286e19ea42f587b344ee6865", "other_bad_hash"}), false},
			},
		},
		{
			name:       "multi_vector_attack",
			expression: `(cf.waf.score.sqli < 40 or cf.waf.score.xss < 40 or cf.waf.score.rce < 40) and cf.threat_score > 20`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"xss_risk_high_threat", NewExecutionContext().SetIntField("cf.waf.score.sqli", 80).SetIntField("cf.waf.score.xss", 25).SetIntField("cf.waf.score.rce", 90).SetIntField("cf.threat_score", 35), true},
				{"all_safe_high_threat", NewExecutionContext().SetIntField("cf.waf.score.sqli", 80).SetIntField("cf.waf.score.xss", 80).SetIntField("cf.waf.score.rce", 80).SetIntField("cf.threat_score", 35), false},
				{"xss_risk_low_threat", NewExecutionContext().SetIntField("cf.waf.score.sqli", 80).SetIntField("cf.waf.score.xss", 25).SetIntField("cf.waf.score.rce", 90).SetIntField("cf.threat_score", 10), false},
			},
		},
		{
			name:       "rate_limit_suspicious",
			expression: `cf.waf.score < 50 and cf.bot_management.score < 50`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"both_suspicious", NewExecutionContext().SetIntField("cf.waf.score", 40).SetIntField("cf.bot_management.score", 35), true},
				{"waf_ok", NewExecutionContext().SetIntField("cf.waf.score", 60).SetIntField("cf.bot_management.score", 35), false},
				{"bot_ok", NewExecutionContext().SetIntField("cf.waf.score", 40).SetIntField("cf.bot_management.score", 60), false},
			},
		},
		{
			name:       "sensitive_path_protection",
			expression: `(http.request.uri.path contains "/admin" or http.request.uri.path contains "/config" or http.request.uri.path contains "/.env") and (cf.waf.score < 80 or cf.threat_score > 10)`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"admin_low_waf", NewExecutionContext().SetStringField("http.request.uri.path", "/admin/settings").SetIntField("cf.waf.score", 70).SetIntField("cf.threat_score", 5), true},
				{"config_high_threat", NewExecutionContext().SetStringField("http.request.uri.path", "/config/db").SetIntField("cf.waf.score", 90).SetIntField("cf.threat_score", 15), true},
				{"admin_safe", NewExecutionContext().SetStringField("http.request.uri.path", "/admin/settings").SetIntField("cf.waf.score", 90).SetIntField("cf.threat_score", 5), false},
				{"normal_path", NewExecutionContext().SetStringField("http.request.uri.path", "/api/users").SetIntField("cf.waf.score", 70).SetIntField("cf.threat_score", 5), false},
			},
		},
		{
			name:       "verified_bot_waf_check",
			expression: `cf.bot_management.verified_bot and cf.waf.score < 30`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"verified_low_waf", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", true).SetIntField("cf.waf.score", 20), true},
				{"verified_high_waf", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", true).SetIntField("cf.waf.score", 50), false},
				{"unverified_low_waf", NewExecutionContext().SetBoolField("cf.bot_management.verified_bot", false).SetIntField("cf.waf.score", 20), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareOperatorAliases tests operator aliases from Cloudflare docs.
// Reference: https://developers.cloudflare.com/ruleset-engine/rules-language/operators/
func TestCloudflareOperatorAliases(t *testing.T) {
	schema := NewSchema().
		AddField("http.request.uri.path", TypeString).
		AddField("http.host", TypeString).
		AddField("ip.src", TypeIP).
		AddField("cf.edge.server_port", TypeInt)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		// From: https://developers.cloudflare.com/ruleset-engine/rules-language/operators/
		// Original: http.request.uri.path eq "/articles/2008/"
		{
			name:       "eq_operator",
			expression: `http.request.uri.path == "/articles/2008/"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2008/"), true},
				{"no_match", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2007/"), false},
			},
		},
		// Original: ip.src ne 203.0.113.0
		{
			name:       "ne_operator",
			expression: `ip.src != 203.0.113.0`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"different", NewExecutionContext().SetIPField("ip.src", "192.168.1.1"), true},
				{"same", NewExecutionContext().SetIPField("ip.src", "203.0.113.0"), false},
			},
		},
		// From: https://developers.cloudflare.com/ruleset-engine/rules-language/expressions/
		// Original: http.host eq "www.example.com" and not cf.edge.server_port in {80 443}
		{
			name:       "eq_and_not_in_set",
			expression: `http.host == "www.example.com" and not cf.edge.server_port in {80, 443}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"host_match_port_8080", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIntField("cf.edge.server_port", 8080), true},
				{"host_match_port_80", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIntField("cf.edge.server_port", 80), false},
				{"host_match_port_443", NewExecutionContext().SetStringField("http.host", "www.example.com").SetIntField("cf.edge.server_port", 443), false},
				{"host_no_match", NewExecutionContext().SetStringField("http.host", "other.com").SetIntField("cf.edge.server_port", 8080), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareWordPressProtection tests WordPress protection expression from Cloudflare docs.
// Reference: https://developers.cloudflare.com/firewall/api/cf-filters/what-is-a-filter/
func TestCloudflareWordPressProtection(t *testing.T) {
	schema := NewSchema().
		AddField("http.request.uri.path", TypeString).
		AddField("ip.src", TypeIP)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		// From: https://developers.cloudflare.com/firewall/api/cf-filters/what-is-a-filter/
		// Original: (http.request.uri.path ~ "^.*wp-login.php$" or http.request.uri.path ~ "^.*xmlrpc.php$") and ip.src ne 93.184.216.34
		{
			name:       "wordpress_brute_force_protection",
			expression: `(http.request.uri.path ~ "^.*wp-login.php$" or http.request.uri.path ~ "^.*xmlrpc.php$") and ip.src != 93.184.216.34`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"wp_login_blocked_ip", NewExecutionContext().SetStringField("http.request.uri.path", "/wp-login.php").SetIPField("ip.src", "192.168.1.1"), true},
				{"xmlrpc_blocked_ip", NewExecutionContext().SetStringField("http.request.uri.path", "/xmlrpc.php").SetIPField("ip.src", "10.0.0.1"), true},
				{"wp_login_allowed_ip", NewExecutionContext().SetStringField("http.request.uri.path", "/wp-login.php").SetIPField("ip.src", "93.184.216.34"), false},
				{"normal_path", NewExecutionContext().SetStringField("http.request.uri.path", "/index.php").SetIPField("ip.src", "192.168.1.1"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareRegexMatches tests regex matching expressions from Cloudflare docs.
// Reference: https://developers.cloudflare.com/ruleset-engine/rules-language/operators/
func TestCloudflareRegexMatches(t *testing.T) {
	schema := NewSchema().
		AddField("http.request.uri.path", TypeString).
		AddField("http.host", TypeString)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		// From: https://developers.cloudflare.com/ruleset-engine/rules-language/operators/
		{
			name:       "matches_year_range",
			expression: `http.request.uri.path matches "^/articles/200[7-8]/$"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"2007", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2007/"), true},
				{"2008", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2008/"), true},
				{"2006", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2006/"), false},
				{"2009", NewExecutionContext().SetStringField("http.request.uri.path", "/articles/2009/"), false},
			},
		},
		// From: https://developers.cloudflare.com/ruleset-engine/rules-language/expressions/
		{
			name:       "autodiscover_regex",
			expression: `http.request.uri.path matches "/autodiscover\.(xml|src)$"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"xml", NewExecutionContext().SetStringField("http.request.uri.path", "/autodiscover.xml"), true},
				{"src", NewExecutionContext().SetStringField("http.request.uri.path", "/autodiscover.src"), true},
				{"txt", NewExecutionContext().SetStringField("http.request.uri.path", "/autodiscover.txt"), false},
			},
		},
		{
			name:       "host_regex_subdomains",
			expression: `http.host matches "^(www|store|blog)\.example\.com"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"www", NewExecutionContext().SetStringField("http.host", "www.example.com"), true},
				{"store", NewExecutionContext().SetStringField("http.host", "store.example.com"), true},
				{"blog", NewExecutionContext().SetStringField("http.host", "blog.example.com"), true},
				{"api", NewExecutionContext().SetStringField("http.host", "api.example.com"), false},
				{"root", NewExecutionContext().SetStringField("http.host", "example.com"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareIPSets tests IP address sets from Cloudflare docs.
// Reference: https://developers.cloudflare.com/ruleset-engine/rules-language/operators/
func TestCloudflareIPSets(t *testing.T) {
	schema := NewSchema().
		AddField("ip.src", TypeIP)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		// From: https://developers.cloudflare.com/ruleset-engine/rules-language/operators/
		{
			name:       "ip_in_set",
			expression: `ip.src in {203.0.113.0, 203.0.113.1}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"first_ip", NewExecutionContext().SetIPField("ip.src", "203.0.113.0"), true},
				{"second_ip", NewExecutionContext().SetIPField("ip.src", "203.0.113.1"), true},
				{"other_ip", NewExecutionContext().SetIPField("ip.src", "203.0.113.2"), false},
			},
		},
		// From: https://developers.cloudflare.com/ruleset-engine/rules-language/values/
		{
			name:       "ip_mixed_set_with_cidr",
			expression: `ip.src in {198.51.100.1, 192.0.2.0/24, 2001:db8::/32}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"exact_ip", NewExecutionContext().SetIPField("ip.src", "198.51.100.1"), true},
				{"in_cidr", NewExecutionContext().SetIPField("ip.src", "192.0.2.100"), true},
				{"in_ipv6_cidr", NewExecutionContext().SetIPField("ip.src", "2001:db8::1"), true},
				{"outside", NewExecutionContext().SetIPField("ip.src", "10.0.0.1"), false},
			},
		},
		// From: https://developers.cloudflare.com/ruleset-engine/rules-language/operators/
		{
			name:       "not_ip_in_cidr",
			expression: `not ip.src in {11.22.33.0/24}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"outside", NewExecutionContext().SetIPField("ip.src", "10.0.0.1"), true},
				{"inside", NewExecutionContext().SetIPField("ip.src", "11.22.33.100"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareComplexExpressions tests complex nested expressions from Cloudflare docs.
// Reference: https://developers.cloudflare.com/ruleset-engine/rules-language/operators/
func TestCloudflareComplexExpressions(t *testing.T) {
	schema := NewSchema().
		AddField("http.host", TypeString).
		AddField("http.request.uri.path", TypeString).
		AddField("ip.src", TypeIP).
		AddField("ip.src.country", TypeString).
		AddField("ip.src.asnum", TypeInt)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		// From: https://developers.cloudflare.com/ruleset-engine/rules-language/operators/
		// Original: http.host eq "api.example.com" and http.request.uri.path eq "/api/v2/auth"
		{
			name:       "api_auth_protection",
			expression: `http.host == "api.example.com" and http.request.uri.path == "/api/v2/auth"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.host", "api.example.com").SetStringField("http.request.uri.path", "/api/v2/auth"), true},
				{"wrong_host", NewExecutionContext().SetStringField("http.host", "www.example.com").SetStringField("http.request.uri.path", "/api/v2/auth"), false},
				{"wrong_path", NewExecutionContext().SetStringField("http.host", "api.example.com").SetStringField("http.request.uri.path", "/api/v1/auth"), false},
			},
		},
		{
			name:       "wp_login_subdomains",
			expression: `http.host matches "^(www|store|blog)\.example\.com" and http.request.uri.path contains "wp-login.php"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"www_wp_login", NewExecutionContext().SetStringField("http.host", "www.example.com").SetStringField("http.request.uri.path", "/wp-login.php"), true},
				{"store_wp_login", NewExecutionContext().SetStringField("http.host", "store.example.com").SetStringField("http.request.uri.path", "/subdir/wp-login.php"), true},
				{"api_wp_login", NewExecutionContext().SetStringField("http.host", "api.example.com").SetStringField("http.request.uri.path", "/wp-login.php"), false},
				{"www_normal", NewExecutionContext().SetStringField("http.host", "www.example.com").SetStringField("http.request.uri.path", "/index.php"), false},
			},
		},
		{
			name:       "country_block",
			expression: `ip.src.country in {"CN", "TH", "US", "ID", "KR", "MY", "IT", "SG", "GB"}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"china", NewExecutionContext().SetStringField("ip.src.country", "CN"), true},
				{"usa", NewExecutionContext().SetStringField("ip.src.country", "US"), true},
				{"germany", NewExecutionContext().SetStringField("ip.src.country", "DE"), false},
				{"france", NewExecutionContext().SetStringField("ip.src.country", "FR"), false},
			},
		},
		{
			name:       "asn_block",
			expression: `ip.src.asnum in {12345, 54321, 11111}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"asn_12345", NewExecutionContext().SetIntField("ip.src.asnum", 12345), true},
				{"asn_54321", NewExecutionContext().SetIntField("ip.src.asnum", 54321), true},
				{"asn_99999", NewExecutionContext().SetIntField("ip.src.asnum", 99999), false},
			},
		},
		// Complex expression from Cloudflare docs
		// Original: ((http.host eq "api.example.com" and http.request.uri.path eq "/api/v2/auth") or (...) or ip.src.country in {"CN" "TH" "US"}) and not ip.src in {11.22.33.0/24}
		{
			name:       "complex_or_and_not",
			expression: `((http.host == "api.example.com" and http.request.uri.path == "/api/v2/auth") or (http.host matches "^(www|store|blog)\.example\.com" and http.request.uri.path contains "wp-login.php") or ip.src.country in {"CN", "TH", "US"}) and not ip.src in {11.22.33.0/24}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"api_auth_blocked", NewExecutionContext().SetStringField("http.host", "api.example.com").SetStringField("http.request.uri.path", "/api/v2/auth").SetIPField("ip.src", "10.0.0.1").SetStringField("ip.src.country", "DE"), true},
				{"wp_login_blocked", NewExecutionContext().SetStringField("http.host", "www.example.com").SetStringField("http.request.uri.path", "/wp-login.php").SetIPField("ip.src", "10.0.0.1").SetStringField("ip.src.country", "DE"), true},
				{"country_blocked", NewExecutionContext().SetStringField("http.host", "other.com").SetStringField("http.request.uri.path", "/index.html").SetIPField("ip.src", "10.0.0.1").SetStringField("ip.src.country", "CN"), true},
				{"whitelisted_ip", NewExecutionContext().SetStringField("http.host", "api.example.com").SetStringField("http.request.uri.path", "/api/v2/auth").SetIPField("ip.src", "11.22.33.100").SetStringField("ip.src.country", "DE"), false},
				{"no_match", NewExecutionContext().SetStringField("http.host", "other.com").SetStringField("http.request.uri.path", "/index.html").SetIPField("ip.src", "10.0.0.1").SetStringField("ip.src.country", "DE"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflareRawStrings tests raw string syntax from Cloudflare docs.
// Reference: https://developers.cloudflare.com/ruleset-engine/rules-language/values/
func TestCloudflareRawStrings(t *testing.T) {
	schema := NewSchema().
		AddField("http.request.uri.path", TypeString)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		// From: https://developers.cloudflare.com/ruleset-engine/rules-language/values/
		{
			name:       "raw_string_regex",
			expression: `http.request.uri.path matches r"/api/login\.aspx$"`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"match", NewExecutionContext().SetStringField("http.request.uri.path", "/api/login.aspx"), true},
				{"no_match", NewExecutionContext().SetStringField("http.request.uri.path", "/api/loginXaspx"), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}

// TestCloudflarePortSets tests port range sets from Cloudflare docs.
// Reference: https://developers.cloudflare.com/ruleset-engine/rules-language/values/
func TestCloudflarePortSets(t *testing.T) {
	schema := NewSchema().
		AddField("tcp.dstport", TypeInt)

	tests := []struct {
		name       string
		expression string
		cases      []struct {
			name     string
			ctx      *ExecutionContext
			expected bool
		}
	}{
		// From: https://developers.cloudflare.com/ruleset-engine/rules-language/values/
		{
			name:       "port_ranges",
			expression: `tcp.dstport in {8000..8009, 8080..8089}`,
			cases: []struct {
				name     string
				ctx      *ExecutionContext
				expected bool
			}{
				{"port_8000", NewExecutionContext().SetIntField("tcp.dstport", 8000), true},
				{"port_8005", NewExecutionContext().SetIntField("tcp.dstport", 8005), true},
				{"port_8009", NewExecutionContext().SetIntField("tcp.dstport", 8009), true},
				{"port_8080", NewExecutionContext().SetIntField("tcp.dstport", 8080), true},
				{"port_8085", NewExecutionContext().SetIntField("tcp.dstport", 8085), true},
				{"port_8010", NewExecutionContext().SetIntField("tcp.dstport", 8010), false},
				{"port_80", NewExecutionContext().SetIntField("tcp.dstport", 80), false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Compile(tt.expression, schema)
			require.NoError(t, err, "compile error for: %s", tt.expression)

			for _, tc := range tt.cases {
				t.Run(tc.name, func(t *testing.T) {
					result, err := filter.Execute(tc.ctx)
					require.NoError(t, err, "execute error")
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	}
}
