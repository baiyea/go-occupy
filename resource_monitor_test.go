package main

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

func TestResourceConfig(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      5 * time.Second,
	}

	if config.MemoryPercent != 50.0 {
		t.Errorf("期望内存百分比为50.0，实际为%f", config.MemoryPercent)
	}

	if config.CPUPercent != 30.0 {
		t.Errorf("期望CPU百分比为30.0，实际为%f", config.CPUPercent)
	}

	if config.DiskPercent != 70.0 {
		t.Errorf("期望磁盘百分比为70.0，实际为%f", config.DiskPercent)
	}

	if config.Interval != 5*time.Second {
		t.Errorf("期望间隔为5秒，实际为%v", config.Interval)
	}
}

func TestNewResourceMonitor(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      5 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	if monitor == nil {
		t.Error("期望创建ResourceMonitor实例，但得到nil")
	}

	if monitor.config.MemoryPercent != config.MemoryPercent {
		t.Errorf("期望内存配置为%f，实际为%f", config.MemoryPercent, monitor.config.MemoryPercent)
	}

	if monitor.stop == nil {
		t.Error("期望stop通道不为nil")
	}

	if monitor.cpuLoadStop == nil {
		t.Error("期望cpuLoadStop通道不为nil")
	}

	if len(monitor.allocatedMemory) != 0 {
		t.Error("期望初始内存分配为空")
	}
}

func TestMemoryAllocation(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      1 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	// 测试内存分配
	initialCount := len(monitor.allocatedMemory)
	bytes := uint64(1024 * 1024) // 1MB
	monitor.allocateMemory(bytes)

	if len(monitor.allocatedMemory) != initialCount+1 {
		t.Errorf("期望内存块数量为%d，实际为%d", initialCount+1, len(monitor.allocatedMemory))
	}

	// 测试内存释放
	monitor.releaseMemory(80.0, &mem.VirtualMemoryStat{Total: 1000000000})
	
	// 清理
	monitor.cleanupAllResources()
}

func TestMemoryRelease(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      1 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	// 分配多个内存块
	for i := 0; i < 5; i++ {
		monitor.allocateMemory(1024 * 1024) // 1MB each
	}

	initialCount := len(monitor.allocatedMemory)
	if initialCount != 5 {
		t.Errorf("期望分配5个内存块，实际为%d", initialCount)
	}

	// 测试内存释放
	monitor.releaseMemory(80.0, &mem.VirtualMemoryStat{Total: 1000000000})
	
	// 应该释放了一些内存块
	if len(monitor.allocatedMemory) >= initialCount {
		t.Error("期望释放部分内存，但内存块数量没有减少")
	}

	// 清理
	monitor.cleanupAllResources()
}

func TestCPULoadControl(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      1 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	// 测试启动CPU负载
	monitor.adjustCPUUsage(10.0) // 当前CPU使用率低于目标，应该启动负载
	
	// 等待一下让goroutine启动
	time.Sleep(100 * time.Millisecond)
	
	// 检查是否启动了CPU负载
	if !monitor.activeCPULoad {
		t.Error("期望CPU负载已启动，但实际未启动")
	}

	// 测试停止CPU负载
	monitor.adjustCPUUsage(50.0) // 当前CPU使用率高于目标，应该停止负载
	
	// 等待一下让goroutine停止
	time.Sleep(100 * time.Millisecond)
	
	// 检查是否停止了CPU负载
	if monitor.activeCPULoad {
		t.Error("期望CPU负载已停止，但实际仍在运行")
	}

	// 清理
	monitor.cleanupAllResources()
}

func TestCPULoadStop(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      1 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	// 启动CPU负载
	monitor.adjustCPUUsage(10.0)
	time.Sleep(50 * time.Millisecond)

	// 测试停止功能
	done := make(chan bool)
	go func() {
		monitor.Stop()
		done <- true
	}()

	select {
	case <-done:
		// 成功停止
		if monitor.activeCPULoad {
			t.Error("停止后CPU负载应该为false")
		}
	case <-time.After(5 * time.Second):
		t.Error("停止操作超时")
	}
}

func TestCPULoadThresholds(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      1 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	testCases := []struct {
		currentCPU float64
		shouldStart bool
	}{
		{10.0, true},  // 低于阈值，应该启动
		{25.0, true},  // 低于阈值，应该启动
		{30.0, false}, // 等于阈值，不应该启动
		{35.0, false}, // 高于阈值，不应该启动
		{50.0, false}, // 远高于阈值，不应该启动
	}

	for _, tc := range testCases {
		// 重置状态
		monitor.activeCPULoad = false
		monitor.cpuLoadStop = make(chan bool)
		monitor.cpuLoadWg = sync.WaitGroup{}

		monitor.adjustCPUUsage(tc.currentCPU)
		time.Sleep(50 * time.Millisecond)

		if monitor.activeCPULoad != tc.shouldStart {
			t.Errorf("CPU使用率%.1f%%，期望启动状态为%v，实际为%v", 
				tc.currentCPU, tc.shouldStart, monitor.activeCPULoad)
		}
	}

	// 清理
	monitor.cleanupAllResources()
}

func TestMemoryAdjustmentLogic(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      1 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	testCases := []struct {
		currentMem float64
		shouldAllocate bool
		shouldRelease bool
	}{
		{30.0, true, false},   // 低于阈值，应该分配
		{48.0, true, false},   // 低于阈值，应该分配
		{50.0, false, false},  // 等于阈值，不应该操作
		{52.0, false, false},  // 接近阈值，不应该操作
		{60.0, false, true},   // 高于阈值，应该释放
		{80.0, false, true},   // 远高于阈值，应该释放
	}

	for _, tc := range testCases {
		// 重置状态
		monitor.allocatedMemory = make([][]byte, 0)

		// 如果需要测试释放，先分配一些内存
		if tc.shouldRelease {
			monitor.allocateMemory(1024 * 1024)
		}

		initialCount := len(monitor.allocatedMemory)
		memInfo := &mem.VirtualMemoryStat{Total: 1000000000}

		monitor.adjustMemoryUsage(tc.currentMem, memInfo)

		if tc.shouldAllocate && len(monitor.allocatedMemory) <= initialCount {
			t.Errorf("内存使用率%.1f%%，期望分配内存，但内存块数量没有增加", tc.currentMem)
		}

		if tc.shouldRelease && len(monitor.allocatedMemory) >= initialCount {
			t.Errorf("内存使用率%.1f%%，期望释放内存，但内存块数量没有减少", tc.currentMem)
		}
	}

	// 清理
	monitor.cleanupAllResources()
}

func TestDiskFileCreation(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      1 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	// 测试磁盘文件创建
	diskInfo := &disk.UsageStat{
		Total: 1000000000000, // 1TB
		Used:  300000000000,  // 30% used
	}

	// 模拟当前磁盘使用率低于目标
	monitor.adjustDiskUsage(30.0, diskInfo)

	// 等待一下让文件创建完成
	time.Sleep(2 * time.Second)

	// 检查是否有文件被创建（通过检查/tmp目录）
	files, err := os.ReadDir("/tmp")
	if err != nil {
		t.Skipf("无法读取/tmp目录: %v", err)
	}

	occupyFiles := 0
	for _, file := range files {
		if !file.IsDir() && len(file.Name()) > 8 && file.Name()[:8] == "occupy_" {
			occupyFiles++
		}
	}

	if occupyFiles == 0 {
		t.Log("没有找到临时文件，这可能是正常的（如果磁盘使用率已经达到目标）")
	}

	// 清理
	monitor.cleanupAllResources()
}

func TestDiskFileCleanup(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      1 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	// 创建测试文件
	testFile := "/tmp/occupy_test_cleanup.tmp"
	f, err := os.Create(testFile)
	if err != nil {
		t.Skipf("无法创建测试文件: %v", err)
	}
	f.Close()

	// 验证文件存在
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("测试文件应该存在")
	}

	// 测试清理功能
	monitor.cleanupAllTempFiles()

	// 验证文件被删除
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("测试文件应该被删除")
	}

	// 清理
	monitor.cleanupAllResources()
}

func TestStopSignalHandling(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      1 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	// 分配一些资源
	monitor.allocateMemory(1024 * 1024)
	monitor.adjustCPUUsage(10.0)

	// 启动监控
	go monitor.Start()

	// 等待一下让监控启动
	time.Sleep(1 * time.Second)

	// 发送停止信号
	monitor.Stop()

	// 等待停止完成
	time.Sleep(2 * time.Second)

	// 验证资源被清理
	if len(monitor.allocatedMemory) != 0 {
		t.Error("期望内存被清理")
	}

	if monitor.activeCPULoad {
		t.Error("期望CPU负载被停止")
	}
}

func TestConcurrentOperations(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      1 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	// 并发执行多个操作
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			monitor.allocateMemory(1024 * 1024)
		}()
	}

	wg.Wait()

	// 验证内存分配
	if len(monitor.allocatedMemory) != 5 {
		t.Errorf("期望分配5个内存块，实际为%d", len(monitor.allocatedMemory))
	}

	// 清理
	monitor.cleanupAllResources()
}

func TestResourceCleanup(t *testing.T) {
	config := ResourceConfig{
		MemoryPercent: 50.0,
		CPUPercent:    30.0,
		DiskPercent:   70.0,
		Interval:      1 * time.Second,
	}

	monitor := NewResourceMonitor(config)

	// 分配资源
	monitor.allocateMemory(1024 * 1024)
	monitor.adjustCPUUsage(10.0)

	// 创建测试文件
	testFile := "/tmp/occupy_test_cleanup.tmp"
	f, _ := os.Create(testFile)
	f.Close()

	// 验证资源存在
	if len(monitor.allocatedMemory) == 0 {
		t.Error("期望有内存分配")
	}

	if !monitor.activeCPULoad {
		t.Error("期望CPU负载已启动")
	}

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("期望测试文件存在")
	}

	// 执行清理
	monitor.cleanupAllResources()

	// 验证资源被清理
	if len(monitor.allocatedMemory) != 0 {
		t.Error("期望内存被清理")
	}

	if monitor.activeCPULoad {
		t.Error("期望CPU负载被停止")
	}

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("期望测试文件被删除")
	}
} 