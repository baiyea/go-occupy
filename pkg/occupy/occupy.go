package occupy

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// ResourceConfig 资源配置
type ResourceConfig struct {
	MemoryPercent float64
	CPUPercent    float64
	DiskPercent   float64
	Interval      time.Duration
}

// ResourceMonitor 资源监控器
type ResourceMonitor struct {
	Config ResourceConfig
	stop   chan bool
	cleanupDone chan bool
	
	// CPU负载控制
	cpuLoadMutex sync.Mutex
	cpuLoadStop  chan bool
	cpuLoadWg    sync.WaitGroup
	ActiveCPULoad bool
	targetCPUWorkers int
	currentCPUWorkers int
	
	// 内存管理
	memoryMutex sync.Mutex
	AllocatedMemory [][]byte
	
	// 磁盘文件管理
	diskMutex sync.Mutex
}

// NewResourceMonitor 创建新的资源监控器
func NewResourceMonitor(config ResourceConfig) *ResourceMonitor {
	return &ResourceMonitor{
		Config: config,
		stop:   make(chan bool),
		cleanupDone: make(chan bool),
		AllocatedMemory: make([][]byte, 0),
	}
}

// Start 开始监控资源使用情况
func (rm *ResourceMonitor) Start() {
	log.Printf("开始监控资源使用情况...")
	log.Printf("目标配置: 内存 %.1f%%, CPU %.1f%%, 磁盘 %.1f%%", 
		rm.Config.MemoryPercent, rm.Config.CPUPercent, rm.Config.DiskPercent)

	ticker := time.NewTicker(rm.Config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.monitorAndAdjust()
		case <-rm.stop:
			log.Println("停止监控")
			rm.cleanupAllResources()
			close(rm.cleanupDone)
			return
		}
	}
}

// Stop 停止监控
func (rm *ResourceMonitor) Stop() {
	select {
	case <-rm.stop:
		return
	default:
		close(rm.stop)
	}
	
	// 等待清理完成
	select {
	case <-rm.cleanupDone:
		log.Println("资源清理已完成")
	case <-time.After(60 * time.Second):
		log.Println("清理超时，强制退出")
	}
}

// GetStopChannel 获取停止通道（用于测试）
func (rm *ResourceMonitor) GetStopChannel() chan bool {
	return rm.stop
}

// AdjustDiskUsage 调整磁盘使用（导出用于测试）
func (rm *ResourceMonitor) AdjustDiskUsage(currentPercent float64, diskInfo *disk.UsageStat) {
	rm.adjustDiskUsage(currentPercent, diskInfo)
}

// AdjustCPUUsage 调整CPU使用（导出用于测试）
func (rm *ResourceMonitor) AdjustCPUUsage(currentPercent float64) {
	rm.adjustCPUUsage(currentPercent)
}

// AdjustMemoryUsage 调整内存使用（导出用于测试）
func (rm *ResourceMonitor) AdjustMemoryUsage(currentPercent float64, memInfo *mem.VirtualMemoryStat) {
	rm.adjustMemoryUsage(currentPercent, memInfo)
}

// AllocateMemory 分配内存（导出用于测试）
func (rm *ResourceMonitor) AllocateMemory(bytes uint64) {
	rm.allocateMemory(bytes)
}

// ReleaseMemory 释放内存（导出用于测试）
func (rm *ResourceMonitor) ReleaseMemory(currentPercent float64, memInfo *mem.VirtualMemoryStat) {
	rm.releaseMemory(currentPercent, memInfo)
}

// stopCPULoad 停止CPU负载
func (rm *ResourceMonitor) stopCPULoad() {
	rm.cpuLoadMutex.Lock()
	defer rm.cpuLoadMutex.Unlock()
	
	if rm.ActiveCPULoad {
		log.Println("正在停止CPU负载...")
		rm.stopCPULoadInternal()
		log.Println("CPU负载已停止")
	}
}

// startCPULoad 开始CPU负载（保持兼容性）
func (rm *ResourceMonitor) startCPULoad() {
	rm.adjustCPUWorkers(runtime.NumCPU())
}

// monitorAndAdjust 监控并调整资源使用
func (rm *ResourceMonitor) monitorAndAdjust() {
	select {
	case <-rm.stop:
		return
	default:
	}
	
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("获取内存信息失败: %v", err)
		return
	}

	select {
	case <-rm.stop:
		return
	default:
	}

	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		log.Printf("获取CPU信息失败: %v", err)
		return
	}

	select {
	case <-rm.stop:
		return
	default:
	}

	diskInfo, err := disk.Usage("/")
	if err != nil {
		log.Printf("获取磁盘信息失败: %v", err)
		return
	}

	select {
	case <-rm.stop:
		return
	default:
	}

	currentMemPercent := memInfo.UsedPercent
	currentCPUPercent := cpuPercent[0]
	currentDiskPercent := diskInfo.UsedPercent

	log.Printf("当前使用情况: 内存 %.1f%%, CPU %.1f%%, 磁盘 %.1f%%", 
		currentMemPercent, currentCPUPercent, currentDiskPercent)

	rm.adjustDiskUsage(currentDiskPercent, diskInfo)
	
	select {
	case <-rm.stop:
		return
	default:
	}
	
	time.Sleep(500 * time.Millisecond)
	
	select {
	case <-rm.stop:
		return
	default:
	}
	
	memInfoAfterDisk, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("重新获取内存信息失败: %v", err)
		return
	}
	
	select {
	case <-rm.stop:
		return
	default:
	}
	
	rm.adjustMemoryUsage(memInfoAfterDisk.UsedPercent, memInfoAfterDisk)
	
	select {
	case <-rm.stop:
		return
	default:
	}
	
	rm.adjustCPUUsage(currentCPUPercent)
}

// adjustMemoryUsage 调整内存使用
func (rm *ResourceMonitor) adjustMemoryUsage(currentPercent float64, memInfo *mem.VirtualMemoryStat) {
	if currentPercent < rm.Config.MemoryPercent {
		targetBytes := uint64((rm.Config.MemoryPercent - currentPercent) / 100.0 * float64(memInfo.Total))
		rm.allocateMemory(targetBytes)
	} else if currentPercent > rm.Config.MemoryPercent+5 {
		rm.releaseMemory(currentPercent, memInfo)
	}
}

// releaseMemory 释放内存
func (rm *ResourceMonitor) releaseMemory(currentPercent float64, memInfo *mem.VirtualMemoryStat) {
	rm.memoryMutex.Lock()
	defer rm.memoryMutex.Unlock()
	
	if len(rm.AllocatedMemory) == 0 {
		return
	}
	
	// 计算需要释放的内存
	targetReleaseBytes := uint64((currentPercent - rm.Config.MemoryPercent) / 100.0 * float64(memInfo.Total))
	currentAllocated := rm.getTotalAllocatedMemory()
	
	if targetReleaseBytes > currentAllocated {
		targetReleaseBytes = currentAllocated
	}
	
	// 释放内存
	releasedBytes := uint64(0)
	for i := len(rm.AllocatedMemory) - 1; i >= 0 && releasedBytes < targetReleaseBytes; i-- {
		chunkSize := uint64(len(rm.AllocatedMemory[i]))
		if releasedBytes+chunkSize <= targetReleaseBytes {
			rm.AllocatedMemory = rm.AllocatedMemory[:i]
			releasedBytes += chunkSize
		} else {
			// 部分释放
			remainingBytes := targetReleaseBytes - releasedBytes
			rm.AllocatedMemory[i] = rm.AllocatedMemory[i][:remainingBytes]
			releasedBytes += remainingBytes
		}
	}
	
	log.Printf("释放内存: %d bytes", releasedBytes)
	
	// 强制垃圾回收
	runtime.GC()
}

// adjustCPUUsage 调整CPU使用
func (rm *ResourceMonitor) adjustCPUUsage(currentPercent float64) {
	// 计算目标工作线程数量
	targetWorkers := 0
	tolerance := 5.0 // 容忍度，避免频繁调整
	
	if currentPercent < rm.Config.CPUPercent - tolerance {
		// CPU使用率低于目标，需要增加负载
		// 根据目标CPU使用率计算工作线程数
		targetWorkers = int(rm.Config.CPUPercent / 100.0 * float64(runtime.NumCPU()))
		if targetWorkers < 1 {
			targetWorkers = 1
		}
		if targetWorkers > runtime.NumCPU() {
			targetWorkers = runtime.NumCPU()
		}
	} else if currentPercent > rm.Config.CPUPercent + tolerance {
		// CPU使用率高于目标，减少或停止负载
		targetWorkers = 0
	} else {
		// 在目标范围内，保持当前状态
		return
	}
	
	rm.adjustCPUWorkers(targetWorkers)
}

// adjustCPUWorkers 调整CPU工作线程数量
func (rm *ResourceMonitor) adjustCPUWorkers(targetWorkers int) {
	rm.cpuLoadMutex.Lock()
	defer rm.cpuLoadMutex.Unlock()
	
	if rm.targetCPUWorkers == targetWorkers {
		return // 目标数量没有变化
	}
	
	rm.targetCPUWorkers = targetWorkers
	
	if targetWorkers == 0 {
		// 停止所有CPU负载
		if rm.ActiveCPULoad {
			log.Printf("停止CPU负载 (当前工作线程: %d)", rm.currentCPUWorkers)
			rm.stopCPULoadInternal()
		}
	} else {
		// 启动或调整CPU负载
		if !rm.ActiveCPULoad {
			log.Printf("启动CPU负载 (目标工作线程: %d)", targetWorkers)
			rm.startCPULoadInternal()
		} else if rm.currentCPUWorkers != targetWorkers {
			log.Printf("调整CPU负载 (当前: %d -> 目标: %d)", rm.currentCPUWorkers, targetWorkers)
			// 重启CPU负载以调整线程数
			rm.stopCPULoadInternal()
			rm.startCPULoadInternal()
		}
	}
}

// startCPULoadInternal 内部启动CPU负载方法
func (rm *ResourceMonitor) startCPULoadInternal() {
	if rm.ActiveCPULoad {
		return
	}
	
	rm.ActiveCPULoad = true
	rm.cpuLoadStop = make(chan bool)
	rm.currentCPUWorkers = rm.targetCPUWorkers
	
	// 启动指定数量的CPU worker
	for i := 0; i < rm.targetCPUWorkers; i++ {
		rm.cpuLoadWg.Add(1)
		go rm.cpuWorker(i)
	}
}

// stopCPULoadInternal 内部停止CPU负载方法
func (rm *ResourceMonitor) stopCPULoadInternal() {
	if !rm.ActiveCPULoad {
		return
	}
	
	close(rm.cpuLoadStop)
	
	// 等待所有CPU worker完成
	done := make(chan bool)
	go func() {
		rm.cpuLoadWg.Wait()
		done <- true
	}()
	
	// 设置超时，避免无限等待
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		log.Println("CPU负载停止超时，强制停止")
	}
	
	rm.ActiveCPULoad = false
	rm.currentCPUWorkers = 0
	rm.cpuLoadStop = make(chan bool)
	rm.cpuLoadWg = sync.WaitGroup{}
}

// cpuWorker CPU工作协程
func (rm *ResourceMonitor) cpuWorker(id int) {
	defer rm.cpuLoadWg.Done()
	
	for {
		select {
		case <-rm.cpuLoadStop:
			return
		default:
			// 持续执行CPU密集型计算
			sum := 0.0
			for i := 0; i < 1000000; i++ {
				sum += float64(i) * 3.14159
				sum = sum * 1.001
				
				// 每1000次迭代检查一次停止信号
				if i%1000 == 0 {
					select {
					case <-rm.cpuLoadStop:
						return
					default:
					}
				}
			}
			_ = sum
		}
	}
}

// doCPUWork 执行CPU密集型工作（保留兼容性）
func (rm *ResourceMonitor) doCPUWork() {
	sum := 0.0
	for i := 0; i < 50000; i++ {
		sum += float64(i) * 3.14159
		sum = sum * 1.001
	}
	_ = sum
}

// adjustDiskUsage 调整磁盘使用
func (rm *ResourceMonitor) adjustDiskUsage(currentPercent float64, diskInfo *disk.UsageStat) {
	if currentPercent < rm.Config.DiskPercent {
		targetBytes := uint64((rm.Config.DiskPercent - currentPercent) / 100.0 * float64(diskInfo.Total))
		rm.createTempFiles(targetBytes)
	} else if currentPercent > rm.Config.DiskPercent+5 {
		rm.cleanupTempFiles()
	}
}

// allocateMemory 分配内存
func (rm *ResourceMonitor) allocateMemory(bytes uint64) {
	rm.memoryMutex.Lock()
	defer rm.memoryMutex.Unlock()
	
	chunkSize := uint64(100 * 1024 * 1024) // 100MB per chunk
	remainingBytes := bytes
	
	for remainingBytes > 0 {
		currentChunk := chunkSize
		if remainingBytes < chunkSize {
			currentChunk = remainingBytes
		}
		
		memory := make([]byte, currentChunk)
		for i := range memory {
			memory[i] = byte(i % 256)
		}
		
		rm.AllocatedMemory = append(rm.AllocatedMemory, memory)
		remainingBytes -= currentChunk
		
		log.Printf("分配内存: %d bytes", currentChunk)
	}
}

// createTempFiles 创建临时文件
func (rm *ResourceMonitor) createTempFiles(targetBytes uint64) {
	rm.diskMutex.Lock()
	defer rm.diskMutex.Unlock()
	
	tempDir := os.TempDir()
	if testTempDir := os.Getenv("GO_OCCUPY_TEMP_DIR"); testTempDir != "" {
		tempDir = testTempDir
	}
	
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("创建临时目录失败: %v", err)
		return
	}
	
	fileSize := uint64(5 * 1024 * 1024 * 1024) // 5G per file
	remainingBytes := targetBytes
	fileIndex := 0
	
	for remainingBytes > 0 {
		currentFileSize := fileSize
		if remainingBytes < fileSize {
			currentFileSize = remainingBytes
		}
		
		fileName := fmt.Sprintf("go_occupy_temp_%d_%d.dat", time.Now().Unix(), fileIndex)
		filePath := filepath.Join(tempDir, fileName)
		
		file, err := os.Create(filePath)
		if err != nil {
			log.Printf("创建临时文件失败: %v", err)
			return
		}
		
		data := make([]byte, currentFileSize)
		for i := range data {
			data[i] = byte(i % 256)
		}
		
		if _, err := file.Write(data); err != nil {
			log.Printf("写入临时文件失败: %v", err)
			file.Close()
			os.Remove(filePath)
			return
		}
		
		file.Close()
		log.Printf("创建临时文件: %s (%d bytes)", fileName, currentFileSize)
		
		remainingBytes -= currentFileSize
		fileIndex++
	}
}

// cleanupTempFiles 清理临时文件
func (rm *ResourceMonitor) cleanupTempFiles() {
	rm.diskMutex.Lock()
	defer rm.diskMutex.Unlock()
	
	tempDir := os.TempDir()
	if testTempDir := os.Getenv("GO_OCCUPY_TEMP_DIR"); testTempDir != "" {
		tempDir = testTempDir
	}
	
	pattern := filepath.Join(tempDir, "go_occupy_temp_*.dat")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("查找临时文件失败: %v", err)
		return
	}
	
	deletedCount := 0
	for _, file := range matches {
		if err := os.Remove(file); err != nil {
			log.Printf("删除临时文件失败: %s, %v", file, err)
		} else {
			deletedCount++
		}
	}
	
	if deletedCount > 0 {
		log.Printf("清理临时文件: %d 个", deletedCount)
	}
}

// cleanupMemory 清理内存
func (rm *ResourceMonitor) cleanupMemory() {
	rm.memoryMutex.Lock()
	defer rm.memoryMutex.Unlock()
	
	if len(rm.AllocatedMemory) == 0 {
		return
	}
	
	totalBytes := rm.getTotalAllocatedMemory()
	rm.AllocatedMemory = make([][]byte, 0)
	
	log.Printf("清理内存: %d bytes", totalBytes)
	runtime.GC()
}

// getTotalAllocatedMemory 获取总分配内存
func (rm *ResourceMonitor) getTotalAllocatedMemory() uint64 {
	total := uint64(0)
	for _, memory := range rm.AllocatedMemory {
		total += uint64(len(memory))
	}
	return total
}

// cleanupAllTempFiles 清理所有临时文件
func (rm *ResourceMonitor) cleanupAllTempFiles() {
	tempDir := os.TempDir()
	if testTempDir := os.Getenv("GO_OCCUPY_TEMP_DIR"); testTempDir != "" {
		tempDir = testTempDir
	}
	
	pattern := filepath.Join(tempDir, "go_occupy_temp_*.dat")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("查找临时文件失败: %v", err)
		return
	}
	
	deletedCount := 0
	for _, file := range matches {
		if err := os.Remove(file); err != nil {
			log.Printf("删除临时文件失败: %s, %v", file, err)
		} else {
			deletedCount++
		}
	}
	
	if deletedCount > 0 {
		log.Printf("清理所有临时文件: %d 个", deletedCount)
	}
}

// cleanupAllResources 清理所有资源
func (rm *ResourceMonitor) cleanupAllResources() {
	log.Println("开始清理所有资源...")
	
	// 停止CPU负载
	log.Println("正在停止CPU负载...")
	rm.stopCPULoad()
	
	// 清理内存
	log.Println("正在清理内存...")
	rm.cleanupMemory()
	
	// 清理临时文件
	log.Println("正在清理临时文件...")
	rm.cleanupAllTempFiles()
	
	// 强制垃圾回收
	log.Println("执行垃圾回收...")
	runtime.GC()
	
	log.Println("资源清理完成")
}

// CleanupAllResources 清理所有资源（导出用于测试）
func (rm *ResourceMonitor) CleanupAllResources() {
	rm.cleanupAllResources()
}

// CleanupAllTempFiles 清理所有临时文件（导出用于测试）
func (rm *ResourceMonitor) CleanupAllTempFiles() {
	rm.cleanupAllTempFiles()
} 