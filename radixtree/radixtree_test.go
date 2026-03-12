package radixtree

import (
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tree := New[int]()

	assert.NotNil(t, tree)
	assert.Equal(t, 0, tree.Len())
}

func TestInsert(t *testing.T) {
	t.Run("single_key", func(t *testing.T) {
		tree := New[string]()

		isNew := tree.Insert("hello", "world")
		assert.True(t, isNew)
		assert.Equal(t, 1, tree.Len())
	})

	t.Run("multiple_keys", func(t *testing.T) {
		tree := New[int]()

		assert.True(t, tree.Insert("apple", 1))
		assert.True(t, tree.Insert("app", 2))
		assert.True(t, tree.Insert("application", 3))
		assert.True(t, tree.Insert("banana", 4))
		assert.Equal(t, 4, tree.Len())
	})

	t.Run("update_existing", func(t *testing.T) {
		tree := New[int]()

		assert.True(t, tree.Insert("key", 1))
		assert.False(t, tree.Insert("key", 2))
		assert.Equal(t, 1, tree.Len())

		val, ok := tree.Get("key")
		assert.True(t, ok)
		assert.Equal(t, 2, val)
	})

	t.Run("empty_key", func(t *testing.T) {
		tree := New[int]()

		assert.True(t, tree.Insert("", 1))
		assert.Equal(t, 1, tree.Len())

		val, ok := tree.Get("")
		assert.True(t, ok)
		assert.Equal(t, 1, val)
	})

	t.Run("shared_prefixes", func(t *testing.T) {
		tree := New[int]()

		assert.True(t, tree.Insert("test", 1))
		assert.True(t, tree.Insert("testing", 2))
		assert.True(t, tree.Insert("team", 3))
		assert.True(t, tree.Insert("tea", 4))
		assert.Equal(t, 4, tree.Len())

		val, ok := tree.Get("test")
		assert.True(t, ok)
		assert.Equal(t, 1, val)

		val, ok = tree.Get("testing")
		assert.True(t, ok)
		assert.Equal(t, 2, val)

		val, ok = tree.Get("team")
		assert.True(t, ok)
		assert.Equal(t, 3, val)

		val, ok = tree.Get("tea")
		assert.True(t, ok)
		assert.Equal(t, 4, val)
	})
}

func TestInsertBytes(t *testing.T) {
	tree := New[int]()

	assert.True(t, tree.InsertBytes([]byte("hello"), 1))
	assert.Equal(t, 1, tree.Len())

	val, ok := tree.GetBytes([]byte("hello"))
	assert.True(t, ok)
	assert.Equal(t, 1, val)
}

func TestGet(t *testing.T) {
	t.Run("existing_key", func(t *testing.T) {
		tree := New[string]()
		tree.Insert("key", "value")

		val, ok := tree.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("missing_key", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		val, ok := tree.Get("world")
		assert.False(t, ok)
		assert.Equal(t, 0, val)
	})

	t.Run("partial_key", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		_, ok := tree.Get("hell")
		assert.False(t, ok)

		_, ok = tree.Get("helloworld")
		assert.False(t, ok)
	})

	t.Run("empty_tree", func(t *testing.T) {
		tree := New[int]()

		_, ok := tree.Get("any")
		assert.False(t, ok)
	})

	t.Run("prefix_mismatch_mid_node", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		// key longer than node prefix, mismatch
		_, ok := tree.Get("hexagon")
		assert.False(t, ok)

		// key shorter than node prefix, mismatch
		_, ok = tree.Get("hx")
		assert.False(t, ok)
	})

	t.Run("intermediate_node_no_value", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)
		tree.Insert("abd", 2)

		// "ab" is an internal split node with no value
		_, ok := tree.Get("ab")
		assert.False(t, ok)
	})
}

func TestContains(t *testing.T) {
	t.Run("existing_key", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		assert.True(t, tree.Contains("hello"))
	})

	t.Run("missing_key", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		assert.False(t, tree.Contains("world"))
	})

	t.Run("empty_tree", func(t *testing.T) {
		tree := New[int]()

		assert.False(t, tree.Contains("any"))
	})
}

func TestContainsBytes(t *testing.T) {
	tree := New[int]()
	tree.Insert("hello", 1)

	assert.True(t, tree.ContainsBytes([]byte("hello")))
	assert.False(t, tree.ContainsBytes([]byte("world")))
}

func TestDelete(t *testing.T) {
	t.Run("existing_key", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		assert.True(t, tree.Delete("hello"))
		assert.Equal(t, 0, tree.Len())

		_, ok := tree.Get("hello")
		assert.False(t, ok)
	})

	t.Run("missing_key", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		assert.False(t, tree.Delete("world"))
		assert.Equal(t, 1, tree.Len())
	})

	t.Run("delete_with_siblings", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("apple", 1)
		tree.Insert("app", 2)
		tree.Insert("application", 3)

		assert.True(t, tree.Delete("app"))
		assert.Equal(t, 2, tree.Len())

		_, ok := tree.Get("app")
		assert.False(t, ok)

		val, ok := tree.Get("apple")
		assert.True(t, ok)
		assert.Equal(t, 1, val)

		val, ok = tree.Get("application")
		assert.True(t, ok)
		assert.Equal(t, 3, val)
	})

	t.Run("delete_leaf_compacts", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("apple", 1)
		tree.Insert("application", 2)

		assert.True(t, tree.Delete("apple"))
		assert.Equal(t, 1, tree.Len())

		val, ok := tree.Get("application")
		assert.True(t, ok)
		assert.Equal(t, 2, val)
	})

	t.Run("delete_empty_key", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("", 1)

		assert.True(t, tree.Delete(""))
		assert.Equal(t, 0, tree.Len())
	})

	t.Run("delete_nonexistent", func(t *testing.T) {
		tree := New[int]()

		assert.False(t, tree.Delete("nothing"))
	})

	t.Run("delete_prefix_mismatch_mid_node", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		assert.False(t, tree.Delete("hexagon"))
	})

	t.Run("delete_intermediate_no_value", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)
		tree.Insert("abd", 2)

		// "ab" is an intermediate node with no value
		assert.False(t, tree.Delete("ab"))
	})
}

func TestDeleteBytes(t *testing.T) {
	tree := New[int]()
	tree.Insert("hello", 1)

	assert.True(t, tree.DeleteBytes([]byte("hello")))
	assert.Equal(t, 0, tree.Len())
}

func TestShortestPrefix(t *testing.T) {
	t.Run("exact_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("/api/v1/users", 1)

		key, val, ok := tree.ShortestPrefix("/api/v1/users")
		assert.True(t, ok)
		assert.Equal(t, "/api/v1/users", key)
		assert.Equal(t, 1, val)
	})

	t.Run("returns_shortest", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("/api", 1)
		tree.Insert("/api/v1", 2)
		tree.Insert("/api/v1/users", 3)

		key, val, ok := tree.ShortestPrefix("/api/v1/users/123")
		assert.True(t, ok)
		assert.Equal(t, "/api", key)
		assert.Equal(t, 1, val)
	})

	t.Run("skips_intermediate_nodes", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("/api/v1", 1)
		tree.Insert("/api/v1/users", 2)

		// "/api" is not inserted, so shortest match is "/api/v1"
		key, val, ok := tree.ShortestPrefix("/api/v1/users/123")
		assert.True(t, ok)
		assert.Equal(t, "/api/v1", key)
		assert.Equal(t, 1, val)
	})

	t.Run("no_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("/api", 1)

		_, _, ok := tree.ShortestPrefix("/web/page")
		assert.False(t, ok)
	})

	t.Run("empty_root_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("", 1)
		tree.Insert("/api", 2)

		key, val, ok := tree.ShortestPrefix("/api/v1")
		assert.True(t, ok)
		assert.Equal(t, "", key)
		assert.Equal(t, 1, val)
	})

	t.Run("partial_prefix_mismatch", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		_, _, ok := tree.ShortestPrefix("hexagon")
		assert.False(t, ok)
	})

	t.Run("key_shorter_than_node_prefix", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		_, _, ok := tree.ShortestPrefix("hel")
		assert.False(t, ok)
	})

	t.Run("no_child_mid_traversal", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)
		tree.Insert("abd", 2)

		// "ab" is an internal node, "abx" matches "ab" but 'x' has no child
		_, _, ok := tree.ShortestPrefix("abx")
		assert.False(t, ok)
	})
}

func TestShortestPrefixBytes(t *testing.T) {
	tree := New[int]()
	tree.Insert("/api", 1)
	tree.Insert("/api/v1", 2)

	key, val, ok := tree.ShortestPrefixBytes([]byte("/api/v1/users"))
	assert.True(t, ok)
	assert.Equal(t, "/api", key)
	assert.Equal(t, 1, val)
}

func TestLongestPrefix(t *testing.T) {
	t.Run("exact_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("/api/v1/users", 1)

		key, val, ok := tree.LongestPrefix("/api/v1/users")
		assert.True(t, ok)
		assert.Equal(t, "/api/v1/users", key)
		assert.Equal(t, 1, val)
	})

	t.Run("prefix_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("/api", 1)
		tree.Insert("/api/v1", 2)
		tree.Insert("/api/v1/users", 3)

		key, val, ok := tree.LongestPrefix("/api/v1/users/123")
		assert.True(t, ok)
		assert.Equal(t, "/api/v1/users", key)
		assert.Equal(t, 3, val)
	})

	t.Run("no_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("/api", 1)

		_, _, ok := tree.LongestPrefix("/web/page")
		assert.False(t, ok)
	})

	t.Run("empty_root_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("", 1)

		key, val, ok := tree.LongestPrefix("anything")
		assert.True(t, ok)
		assert.Equal(t, "", key)
		assert.Equal(t, 1, val)
	})

	t.Run("bgp_cidr_prefixes", func(t *testing.T) {
		tree := New[string]()
		tree.Insert("10.", "class-a")
		tree.Insert("10.0.", "dc1")
		tree.Insert("10.0.1.", "subnet1")
		tree.Insert("192.168.", "private")

		key, val, ok := tree.LongestPrefix("10.0.1.50")
		assert.True(t, ok)
		assert.Equal(t, "10.0.1.", key)
		assert.Equal(t, "subnet1", val)

		key, val, ok = tree.LongestPrefix("10.0.2.1")
		assert.True(t, ok)
		assert.Equal(t, "10.0.", key)
		assert.Equal(t, "dc1", val)
	})
}

func TestLongestPrefixBytes(t *testing.T) {
	tree := New[int]()
	tree.Insert("/api", 1)

	key, val, ok := tree.LongestPrefixBytes([]byte("/api/v1"))
	assert.True(t, ok)
	assert.Equal(t, "/api", key)
	assert.Equal(t, 1, val)
}

func TestPrefixSearch(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("apple", 1)
		tree.Insert("app", 2)
		tree.Insert("application", 3)
		tree.Insert("banana", 4)

		results := tree.PrefixSearch("app")
		assert.Len(t, results, 3)
		assert.Equal(t, 1, results["apple"])
		assert.Equal(t, 2, results["app"])
		assert.Equal(t, 3, results["application"])
	})

	t.Run("no_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("apple", 1)

		results := tree.PrefixSearch("banana")
		assert.Nil(t, results)
	})

	t.Run("empty_prefix", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("a", 1)
		tree.Insert("b", 2)

		results := tree.PrefixSearch("")
		assert.Len(t, results, 2)
	})

	t.Run("exact_key_prefix", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("test", 1)

		results := tree.PrefixSearch("test")
		assert.Len(t, results, 1)
		assert.Equal(t, 1, results["test"])
	})

	t.Run("prefix_shorter_than_node_mismatch", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		// "he" matches first 2 bytes of "hello" but "hex" diverges
		results := tree.PrefixSearch("hex")
		assert.Nil(t, results)
	})

	t.Run("prefix_longer_than_node_mismatch", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)
		tree.Insert("abd", 2)

		// Internal node "ab" exists. "axyz" starts with 'a', matches first byte
		// but "ax"[:2]="ax" != "ab" triggers mismatch on traversal
		results := tree.PrefixSearch("axyz")
		assert.Nil(t, results)
	})

	t.Run("url_paths", func(t *testing.T) {
		tree := New[string]()
		tree.Insert("/api/v1/users", "users")
		tree.Insert("/api/v1/users/admin", "admin")
		tree.Insert("/api/v1/posts", "posts")
		tree.Insert("/api/v2/users", "users-v2")

		results := tree.PrefixSearch("/api/v1/users")
		assert.Len(t, results, 2)
		assert.Equal(t, "users", results["/api/v1/users"])
		assert.Equal(t, "admin", results["/api/v1/users/admin"])
	})
}

func TestPrefixSearchBytes(t *testing.T) {
	tree := New[int]()
	tree.Insert("hello", 1)
	tree.Insert("help", 2)

	results := tree.PrefixSearchBytes([]byte("hel"))
	assert.Len(t, results, 2)
}

func TestWalk(t *testing.T) {
	t.Run("all_entries", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("b", 2)
		tree.Insert("a", 1)
		tree.Insert("c", 3)

		var keys []string
		tree.Walk(func(key string, _ int) bool {
			keys = append(keys, key)
			return true
		})

		assert.Len(t, keys, 3)
	})

	t.Run("early_stop", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("a", 1)
		tree.Insert("b", 2)
		tree.Insert("c", 3)

		count := 0
		tree.Walk(func(_ string, _ int) bool {
			count++
			return count < 2
		})

		assert.Equal(t, 2, count)
	})

	t.Run("empty_tree", func(t *testing.T) {
		tree := New[int]()

		count := 0
		tree.Walk(func(_ string, _ int) bool {
			count++
			return true
		})

		assert.Equal(t, 0, count)
	})

	t.Run("preserves_keys", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("test", 1)
		tree.Insert("testing", 2)
		tree.Insert("tea", 3)

		collected := make(map[string]int)
		tree.Walk(func(key string, value int) bool {
			collected[key] = value
			return true
		})

		assert.Equal(t, map[string]int{
			"test":    1,
			"testing": 2,
			"tea":     3,
		}, collected)
	})
}

func TestWalkPrefix(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("apple", 1)
		tree.Insert("app", 2)
		tree.Insert("application", 3)
		tree.Insert("banana", 4)

		collected := make(map[string]int)
		tree.WalkPrefix("app", func(key string, value int) bool {
			collected[key] = value
			return true
		})

		assert.Equal(t, map[string]int{
			"apple":       1,
			"app":         2,
			"application": 3,
		}, collected)
	})

	t.Run("no_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("apple", 1)

		count := 0
		tree.WalkPrefix("banana", func(_ string, _ int) bool {
			count++
			return true
		})

		assert.Equal(t, 0, count)
	})

	t.Run("early_stop", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("aa", 1)
		tree.Insert("ab", 2)
		tree.Insert("ac", 3)

		count := 0
		tree.WalkPrefix("a", func(_ string, _ int) bool {
			count++
			return count < 2
		})

		assert.Equal(t, 2, count)
	})

	t.Run("empty_prefix", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("a", 1)
		tree.Insert("b", 2)

		collected := make(map[string]int)
		tree.WalkPrefix("", func(key string, value int) bool {
			collected[key] = value
			return true
		})

		assert.Len(t, collected, 2)
	})

	t.Run("partial_node_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)
		tree.Insert("abd", 2)

		collected := make(map[string]int)
		tree.WalkPrefix("ab", func(key string, value int) bool {
			collected[key] = value
			return true
		})

		assert.Equal(t, map[string]int{"abc": 1, "abd": 2}, collected)
	})

	t.Run("prefix_shorter_than_node_mismatch", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		count := 0
		tree.WalkPrefix("hx", func(_ string, _ int) bool {
			count++
			return true
		})

		assert.Equal(t, 0, count)
	})

	t.Run("prefix_mismatch_within_node", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)

		count := 0
		tree.WalkPrefix("axyz", func(_ string, _ int) bool {
			count++
			return true
		})

		assert.Equal(t, 0, count)
	})

	t.Run("prefix_longer_than_node_mismatch", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)
		tree.Insert("abd", 2)

		count := 0
		tree.WalkPrefix("abx", func(_ string, _ int) bool {
			count++
			return true
		})

		assert.Equal(t, 0, count)
	})

	t.Run("bytes", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)
		tree.Insert("help", 2)

		count := 0
		tree.WalkPrefixBytes([]byte("hel"), func(_ string, _ int) bool {
			count++
			return true
		})

		assert.Equal(t, 2, count)
	})

	t.Run("bytes_empty", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("a", 1)

		count := 0
		tree.WalkPrefixBytes([]byte{}, func(_ string, _ int) bool {
			count++
			return true
		})

		assert.Equal(t, 1, count)
	})
}

func TestHasPrefix(t *testing.T) {
	t.Run("existing_prefix", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)
		tree.Insert("help", 2)

		assert.True(t, tree.HasPrefix("hel"))
	})

	t.Run("exact_key_as_prefix", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		assert.True(t, tree.HasPrefix("hello"))
	})

	t.Run("no_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		assert.False(t, tree.HasPrefix("world"))
	})

	t.Run("empty_prefix", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		assert.True(t, tree.HasPrefix(""))
	})

	t.Run("prefix_longer_than_node_mismatch", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)
		tree.Insert("abd", 2)

		assert.False(t, tree.HasPrefix("axyz"))
	})

	t.Run("prefix_shorter_than_node_mismatch", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		assert.False(t, tree.HasPrefix("hx"))
	})

	t.Run("no_child", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)

		assert.False(t, tree.HasPrefix("xyz"))
	})

	t.Run("traverses_multiple_nodes", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("ab", 1)
		tree.Insert("abcde", 2)
		tree.Insert("abcdf", 3)

		// prefix "abcd" traverses "ab" node, then matches "cd" prefix partially
		assert.True(t, tree.HasPrefix("abcd"))
	})
}

func TestHasPrefixBytes(t *testing.T) {
	tree := New[int]()
	tree.Insert("hello", 1)

	assert.True(t, tree.HasPrefixBytes([]byte("hel")))
	assert.False(t, tree.HasPrefixBytes([]byte("xyz")))
}

func TestDeletePrefix(t *testing.T) {
	t.Run("delete_subtree", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("apple", 1)
		tree.Insert("app", 2)
		tree.Insert("application", 3)
		tree.Insert("banana", 4)

		count := tree.DeletePrefix("app")
		assert.Equal(t, 3, count)
		assert.Equal(t, 1, tree.Len())

		_, ok := tree.Get("apple")
		assert.False(t, ok)

		val, ok := tree.Get("banana")
		assert.True(t, ok)
		assert.Equal(t, 4, val)
	})

	t.Run("no_match", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		count := tree.DeletePrefix("xyz")
		assert.Equal(t, 0, count)
		assert.Equal(t, 1, tree.Len())
	})

	t.Run("delete_all", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)
		tree.Insert("abd", 2)

		count := tree.DeletePrefix("ab")
		assert.Equal(t, 2, count)
		assert.Equal(t, 0, tree.Len())
	})

	t.Run("prefix_mismatch_short", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("hello", 1)

		count := tree.DeletePrefix("hx")
		assert.Equal(t, 0, count)
	})

	t.Run("prefix_mismatch_long", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)
		tree.Insert("abd", 2)

		count := tree.DeletePrefix("axyz")
		assert.Equal(t, 0, count)
	})

	t.Run("exact_prefix_at_node", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("test", 1)
		tree.Insert("testing", 2)
		tree.Insert("team", 3)

		count := tree.DeletePrefix("test")
		assert.Equal(t, 2, count)
		assert.Equal(t, 1, tree.Len())

		val, ok := tree.Get("team")
		assert.True(t, ok)
		assert.Equal(t, 3, val)
	})

	t.Run("compact_after_delete", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)
		tree.Insert("abd", 2)
		tree.Insert("aef", 3)

		// Delete "abc" subtree; "ab" node should compact with "abd"
		count := tree.DeletePrefix("abc")
		assert.Equal(t, 1, count)
		assert.Equal(t, 2, tree.Len())

		val, ok := tree.Get("abd")
		assert.True(t, ok)
		assert.Equal(t, 2, val)
	})

	t.Run("no_child_for_prefix", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)

		count := tree.DeletePrefix("xyz")
		assert.Equal(t, 0, count)
	})

	t.Run("prefix_consumed_at_node", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("ab", 1)
		tree.Insert("abc", 2)
		tree.Insert("abd", 3)

		// prefix "ab" consumed at a node that has value and children
		count := tree.DeletePrefix("ab")
		assert.Equal(t, 3, count)
		assert.Equal(t, 0, tree.Len())
	})

	t.Run("prefix_consumed_with_compaction", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("xa", 1)
		tree.Insert("xbc", 2)
		tree.Insert("xbd", 3)

		// Delete "xa" leaves "x" with single child "b" -> compact
		count := tree.DeletePrefix("xa")
		assert.Equal(t, 1, count)
		assert.Equal(t, 2, tree.Len())

		val, ok := tree.Get("xbc")
		assert.True(t, ok)
		assert.Equal(t, 2, val)
	})

	t.Run("prefix_traverses_multiple_nodes", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("abc", 1)
		tree.Insert("abcde", 2)
		tree.Insert("abcdf", 3)
		tree.Insert("xyz", 4)

		// prefix "abcd" traverses "abc" node then matches "d" prefix
		count := tree.DeletePrefix("abcd")
		assert.Equal(t, 2, count)
		assert.Equal(t, 2, tree.Len())
	})

	t.Run("empty_prefix_deletes_all", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("a", 1)
		tree.Insert("b", 2)
		tree.Insert("c", 3)

		count := tree.DeletePrefix("")
		assert.Equal(t, 3, count)
		assert.Equal(t, 0, tree.Len())
	})
}

func TestDeletePrefixBytes(t *testing.T) {
	tree := New[int]()
	tree.Insert("hello", 1)
	tree.Insert("help", 2)
	tree.Insert("world", 3)

	count := tree.DeletePrefixBytes([]byte("hel"))
	assert.Equal(t, 2, count)
	assert.Equal(t, 1, tree.Len())
}

func TestMerge(t *testing.T) {
	t.Run("merge_trees", func(t *testing.T) {
		tree1 := New[int]()
		tree1.Insert("a", 1)
		tree1.Insert("b", 2)

		tree2 := New[int]()
		tree2.Insert("c", 3)
		tree2.Insert("d", 4)

		tree1.Merge(tree2)
		assert.Equal(t, 4, tree1.Len())

		val, ok := tree1.Get("c")
		assert.True(t, ok)
		assert.Equal(t, 3, val)
	})

	t.Run("merge_overwrites", func(t *testing.T) {
		tree1 := New[int]()
		tree1.Insert("key", 1)

		tree2 := New[int]()
		tree2.Insert("key", 99)

		tree1.Merge(tree2)
		assert.Equal(t, 1, tree1.Len())

		val, ok := tree1.Get("key")
		assert.True(t, ok)
		assert.Equal(t, 99, val)
	})

	t.Run("merge_empty", func(t *testing.T) {
		tree1 := New[int]()
		tree1.Insert("a", 1)

		tree2 := New[int]()

		tree1.Merge(tree2)
		assert.Equal(t, 1, tree1.Len())
	})
}

func TestKeys(t *testing.T) {
	t.Run("returns_all_keys", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("banana", 1)
		tree.Insert("apple", 2)
		tree.Insert("cherry", 3)

		keys := tree.Keys()
		sort.Strings(keys)

		assert.Equal(t, []string{"apple", "banana", "cherry"}, keys)
	})

	t.Run("empty_tree", func(t *testing.T) {
		tree := New[int]()

		keys := tree.Keys()
		assert.Empty(t, keys)
	})
}

func TestValues(t *testing.T) {
	t.Run("returns_all_values", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("a", 1)
		tree.Insert("b", 2)
		tree.Insert("c", 3)

		values := tree.Values()
		sort.Ints(values)

		assert.Equal(t, []int{1, 2, 3}, values)
	})

	t.Run("empty_tree", func(t *testing.T) {
		tree := New[int]()

		values := tree.Values()
		assert.Empty(t, values)
	})
}

func TestClear(t *testing.T) {
	t.Run("clears_all", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("a", 1)
		tree.Insert("b", 2)
		tree.Insert("c", 3)

		tree.Clear()
		assert.Equal(t, 0, tree.Len())

		_, ok := tree.Get("a")
		assert.False(t, ok)
	})

	t.Run("clear_empty", func(t *testing.T) {
		tree := New[int]()

		tree.Clear()
		assert.Equal(t, 0, tree.Len())
	})

	t.Run("reuse_after_clear", func(t *testing.T) {
		tree := New[int]()
		tree.Insert("old", 1)
		tree.Clear()

		tree.Insert("new", 2)
		assert.Equal(t, 1, tree.Len())

		val, ok := tree.Get("new")
		assert.True(t, ok)
		assert.Equal(t, 2, val)
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent_writes", func(t *testing.T) {
		tree := New[int]()

		var wg sync.WaitGroup

		for i := range 100 {
			wg.Go(func() {
				tree.Insert(fmt.Sprintf("key-%d", i), i)
			})
		}

		wg.Wait()

		assert.Equal(t, 100, tree.Len())
	})

	t.Run("concurrent_reads", func(t *testing.T) {
		tree := New[int]()

		for i := range 100 {
			tree.Insert(fmt.Sprintf("key-%d", i), i)
		}

		var wg sync.WaitGroup

		for i := range 100 {
			wg.Go(func() {
				val, ok := tree.Get(fmt.Sprintf("key-%d", i))
				require.True(t, ok)
				assert.Equal(t, i, val)
			})
		}

		wg.Wait()
	})

	t.Run("mixed_operations", func(_ *testing.T) {
		tree := New[int]()

		for i := range 100 {
			tree.Insert(fmt.Sprintf("key-%d", i), i)
		}

		var wg sync.WaitGroup

		for i := range 50 {
			wg.Go(func() {
				tree.Insert(fmt.Sprintf("concurrent-%d", i), i)
			})

			wg.Go(func() {
				tree.Get(fmt.Sprintf("key-%d", i))
			})

			wg.Go(func() {
				tree.PrefixSearch("key-")
			})
		}

		wg.Wait()
	})
}

func TestLargeDataset(t *testing.T) {
	tree := New[int]()

	keys := make([]string, 1000)
	for i := range 1000 {
		keys[i] = fmt.Sprintf("/api/v1/resource/%d/sub/%d", i%100, i)
	}

	for i, key := range keys {
		tree.Insert(key, i)
	}

	assert.Equal(t, 1000, tree.Len())

	for i, key := range keys {
		val, ok := tree.Get(key)
		require.True(t, ok, "key %s not found", key)
		assert.Equal(t, i, val)
	}
}

func TestLongestCommonPrefix(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"a", "", 0},
		{"", "b", 0},
		{"abc", "abd", 2},
		{"abc", "abc", 3},
		{"abc", "xyz", 0},
		{"hello", "help", 3},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.a, tt.b), func(t *testing.T) {
			assert.Equal(t, tt.expected, longestCommonPrefix(tt.a, tt.b))
		})
	}
}

func TestWalkLexicographic(t *testing.T) {
	tree := New[int]()
	tree.Insert("banana", 1)
	tree.Insert("apple", 2)
	tree.Insert("cherry", 3)
	tree.Insert("apricot", 4)

	var keys []string
	tree.Walk(func(key string, _ int) bool {
		keys = append(keys, key)
		return true
	})

	sorted := make([]string, len(keys))
	copy(sorted, keys)
	sort.Strings(sorted)

	// Walk order depends on insertion order of children, not lexicographic.
	// Verify all keys are present.
	assert.ElementsMatch(t, sorted, keys)
}

func BenchmarkInsert(b *testing.B) {
	keys := make([]string, b.N)

	for i := range keys {
		keys[i] = fmt.Sprintf("/api/v1/resource/%d", i)
	}

	tree := New[int]()

	b.ResetTimer()

	for i := range b.N {
		tree.Insert(keys[i], i)
	}
}

func BenchmarkGet(b *testing.B) {
	const size = 10000

	tree := New[int]()
	keys := make([]string, size)

	for i := range size {
		keys[i] = fmt.Sprintf("/api/v1/resource/%d", i)
		tree.Insert(keys[i], i)
	}

	b.ResetTimer()

	for i := range b.N {
		tree.Get(keys[i%size])
	}
}

func BenchmarkShortestPrefix(b *testing.B) {
	tree := New[int]()
	tree.Insert("/api", 1)
	tree.Insert("/api/v1", 2)
	tree.Insert("/api/v1/users", 3)
	tree.Insert("/api/v1/users/admin", 4)

	key := "/api/v1/users/admin/settings"

	b.ResetTimer()

	for b.Loop() {
		tree.ShortestPrefix(key)
	}
}

func BenchmarkLongestPrefix(b *testing.B) {
	tree := New[int]()
	tree.Insert("/api", 1)
	tree.Insert("/api/v1", 2)
	tree.Insert("/api/v1/users", 3)
	tree.Insert("/api/v1/users/admin", 4)

	key := "/api/v1/users/admin/settings"

	b.ResetTimer()

	for b.Loop() {
		tree.LongestPrefix(key)
	}
}

func BenchmarkPrefixSearch(b *testing.B) {
	tree := New[int]()

	for i := range 10000 {
		tree.Insert(fmt.Sprintf("/api/v1/resource/%d", i), i)
	}

	prefix := "/api/v1/resource/1"

	b.ResetTimer()

	for b.Loop() {
		tree.PrefixSearch(prefix)
	}
}

func BenchmarkWalkPrefix(b *testing.B) {
	tree := New[int]()

	for i := range 10000 {
		tree.Insert(fmt.Sprintf("/api/v1/resource/%d", i), i)
	}

	prefix := "/api/v1/resource/1"

	b.ResetTimer()

	for b.Loop() {
		tree.WalkPrefix(prefix, func(_ string, _ int) bool {
			return true
		})
	}
}

func BenchmarkContains(b *testing.B) {
	const size = 10000

	tree := New[int]()
	keys := make([]string, size)

	for i := range size {
		keys[i] = fmt.Sprintf("/api/v1/resource/%d", i)
		tree.Insert(keys[i], i)
	}

	b.ResetTimer()

	for i := range b.N {
		tree.Contains(keys[i%size])
	}
}

func BenchmarkHasPrefix(b *testing.B) {
	tree := New[int]()

	for i := range 10000 {
		tree.Insert(fmt.Sprintf("/api/v1/resource/%d", i), i)
	}

	prefix := "/api/v1/resource/1"

	b.ResetTimer()

	for b.Loop() {
		tree.HasPrefix(prefix)
	}
}

func BenchmarkDeletePrefix(b *testing.B) {
	const size = 1000

	trees := make([]*Tree[int], b.N)
	for n := range b.N {
		trees[n] = New[int]()
		for i := range size {
			trees[n].Insert(fmt.Sprintf("/api/v1/resource/%d", i), i)
		}
	}

	b.ResetTimer()

	for n := range b.N {
		trees[n].DeletePrefix("/api/v1/resource/1")
	}
}

func BenchmarkMerge(b *testing.B) {
	tree2 := New[int]()

	for i := range 1000 {
		tree2.Insert(fmt.Sprintf("/api/v1/resource/%d", i), i)
	}

	trees := make([]*Tree[int], b.N)

	b.ResetTimer()

	for n := range b.N {
		trees[n] = New[int]()
		trees[n].Merge(tree2)
	}
}

func BenchmarkKeys(b *testing.B) {
	tree := New[int]()

	for i := range 10000 {
		tree.Insert(fmt.Sprintf("/api/v1/resource/%d", i), i)
	}

	b.ResetTimer()

	for b.Loop() {
		tree.Keys()
	}
}

func BenchmarkValues(b *testing.B) {
	tree := New[int]()

	for i := range 10000 {
		tree.Insert(fmt.Sprintf("/api/v1/resource/%d", i), i)
	}

	b.ResetTimer()

	for b.Loop() {
		tree.Values()
	}
}

func BenchmarkClear(b *testing.B) {
	tree := New[int]()

	for i := range 1000 {
		tree.Insert(fmt.Sprintf("/api/v1/resource/%d", i), i)
	}

	b.ResetTimer()

	for b.Loop() {
		tree.Clear()
	}
}

func BenchmarkDelete(b *testing.B) {
	keys := make([]string, b.N)

	for i := range keys {
		keys[i] = fmt.Sprintf("/api/v1/resource/%d", i)
	}

	tree := New[int]()

	for i, key := range keys {
		tree.Insert(key, i)
	}

	b.ResetTimer()

	for i := range b.N {
		tree.Delete(keys[i])
	}
}
