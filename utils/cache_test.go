package utils

import (
	"sync"
	"testing"
)

// 辅助函数：判断两个切片是否相等（用于 Dump 的比较）
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// 测试 Store 和 Load
func TestLRUStoreAndLoad(t *testing.T) {
	lru := NewLRU(3)

	// 存储新 key
	lru.Store("a", 1)
	lru.Store("b", 2)
	lru.Store("c", 3)

	// 验证存在
	val, ok := lru.Load("a")
	if !ok || val != 1 {
		t.Errorf("Load a 期望 (1, true)，得到 (%v, %v)", val, ok)
	}
	val, ok = lru.Load("b")
	if !ok || val != 2 {
		t.Errorf("Load b 期望 (2, true)，得到 (%v, %v)", val, ok)
	}
	val, ok = lru.Load("c")
	if !ok || val != 3 {
		t.Errorf("Load c 期望 (3, true)，得到 (%v, %v)", val, ok)
	}

	// 不存在
	val, ok = lru.Load("d")
	if ok || val != nil {
		t.Errorf("Load d 期望 (nil, false)，得到 (%v, %v)", val, ok)
	}

	// 更新已有 key (应移动到前端)
	lru.Store("a", 100) // value 应更新？但代码中 Store 若 key 存在只移动，不更新值
	// 验证值未变（因为 Store 只移动，不更新值）
	val, ok = lru.Load("a")
	if !ok || val != 1 {
		t.Errorf("更新 a 后，值应仍为 1，实际 %v", val)
	}
}

// 测试容量限制（淘汰最久未使用）
func TestLRUCapacityEviction(t *testing.T) {
	lru := NewLRU(2)
	lru.Store("a", 1)
	lru.Store("b", 2)
	// 现在容量已满
	lru.Store("c", 3) // 应淘汰 "a"

	_, ok := lru.Load("a")
	if ok {
		t.Error("a 应被淘汰")
	}
	val, ok := lru.Load("b")
	if !ok || val != 2 {
		t.Error("b 应存在")
	}
	val, ok = lru.Load("c")
	if !ok || val != 3 {
		t.Error("c 应存在")
	}

	// 访问 b，使其变为最近使用，再插入 d，淘汰 c
	_, _ = lru.Load("b")
	lru.Store("d", 4)
	_, ok = lru.Load("c")
	if ok {
		t.Error("c 应被淘汰")
	}
	val, ok = lru.Load("b")
	if !ok || val != 2 {
		t.Error("b 应存在")
	}
	val, ok = lru.Load("d")
	if !ok || val != 4 {
		t.Error("d 应存在")
	}
}

// 测试 Delete 功能
func TestLRUDelete(t *testing.T) {
	lru := NewLRU(3)
	lru.Store("a", 1)
	lru.Store("b", 2)
	lru.Store("c", 3)

	// 删除存在的
	deletedKey := ""
	deletedVal := interface{}(nil)
	lru.SetDelCallBackFn(func(key, value interface{}) {
		deletedKey = key.(string)
		deletedVal = value
	})

	lru.Delete("b")
	if deletedKey != "b" || deletedVal != 2 {
		t.Errorf("回调未正确调用：key=%v, value=%v", deletedKey, deletedVal)
	}
	_, ok := lru.Load("b")
	if ok {
		t.Error("b 应被删除")
	}
	// 长度减1
	if lru.Len() != 2 {
		t.Errorf("长度应为 2，实际 %d", lru.Len())
	}

	// 删除不存在的 key（不应触发回调，也不改变长度）
	deletedKey = ""
	deletedVal = nil
	lru.Delete("d")
	if deletedKey != "" || deletedVal != nil {
		t.Error("删除不存在的 key 不应触发回调")
	}
	if lru.Len() != 2 {
		t.Errorf("长度仍应为 2，实际 %d", lru.Len())
	}
}

// 测试 Len 方法（正常和异常）
func TestLRULen(t *testing.T) {
	lru := NewLRU(2)
	lru.Store("a", 1)
	lru.Store("b", 2)
	if lru.Len() != 2 {
		t.Errorf("Len 期望 2，实际 %d", lru.Len())
	}

	// 手动破坏一致性（模拟 bug），但这里无法直接操作，故略
	// 但我们可以测试 Len 内部判断不一致的情况，需要反射或修改内部字段，这里不测试
}

// 测试 Dump 方法（简单检查格式）
func TestLRUDump(t *testing.T) {
	lru := NewLRU(3)
	lru.Store("a", "apple")
	lru.Store("b", "banana")
	lru.Store("c", "cherry")

	dump := lru.Dump()
	// 这里依赖 Str 函数，我们只检查非空
	if dump == "" {
		t.Error("Dump 结果不应为空")
	}
	// 可进一步检查包含的字符串，但 Str 函数未知，所以简单测试
	t.Logf("Dump: \n%s", dump)
}

// 测试并发访问
func TestLRUConcurrency(t *testing.T) {
	lru := NewLRU(100)
	var wg sync.WaitGroup
	// 并发的 Store 和 Load
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := i % 50
			lru.Store(key, i)
			val, ok := lru.Load(key)
			if !ok {
				t.Errorf("并发 Load 失败，key=%d", key)
			}
			// 注意：由于 Store 不会更新值，值可能不是最新，但 Load 能拿到即可
			_ = val
		}(i)
	}
	wg.Wait()

	// 检查长度一致
	if lru.Len() < 0 {
		t.Error("Len 返回 -1，数据不一致")
	}
	// 并发 Delete
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			lru.Delete(i)
		}(i)
	}
	wg.Wait()
	// 最终长度可能不为0，但至少不会 panic
}

// 测试 map 重建逻辑（delMapCount > 3*maxSize）
func TestLRUReconstructMap(t *testing.T) {
	// 设置小容量以快速触发
	lru := NewLRU(2)
	// 填充 2 个元素
	lru.Store("a", 1)
	lru.Store("b", 2)

	// 删除并重建：每次删除 delMapCount++，需要超过 3*maxSize = 6 次
	// 但我们只有 2 个元素，删除后长度变空，再插入再删除，重复
	// 但注意删除不存在的 key 不会增加 delMapCount，只有实际删除才增加。
	// 所以我们需要实际删除元素。
	for i := 0; i < 10; i++ {
		// 每次插入一个新 key，然后立即删除，使缓存至少有一个元素可删
		key := string(rune('a' + i))
		lru.Store(key, i)
		lru.Delete(key)
	}
	// 此时 delMapCount 应该已经 > 6，触发了重建
	// 验证缓存仍然工作
	lru.Store("final", 999)
	val, ok := lru.Load("final")
	if !ok || val != 999 {
		t.Error("重建后缓存仍应正常工作")
	}
	// 检查内部 map 和 list 长度一致
	if lru.Len() != 1 {
		t.Errorf("期望长度 1，实际 %d", lru.Len())
	}
}

// 测试 SetDelCallBackFn 设置回调
func TestSetDelCallBackFn(t *testing.T) {
	lru := NewLRU(2)
	called := false
	lru.SetDelCallBackFn(func(key, value interface{}) {
		called = true
	})
	lru.Store("a", 1)
	lru.Store("b", 2)
	lru.Store("c", 3) // 淘汰 a，触发回调
	if !called {
		t.Error("删除回调未被调用")
	}
}
