package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/spf13/cobra"
)

type ResourceConfig struct {
	MemoryPercent float64
	CPUPercent    float64
	DiskPercent   float64
	Interval      time.Duration
}

type ResourceMonitor struct {
	config ResourceConfig
	stop   chan bool
	
	// CPU负载控制
	cpuLoadMutex sync.Mutex
	cpuLoadStop  chan bool
	cpuLoadWg    sync.WaitGroup
	activeCPULoad bool
	
	// 内存管理
	memoryMutex sync.Mutex
	allocatedMemory [][]byte
	
	// 磁盘文件管理
	diskMutex sync.Mutex
}

func NewResourceMonitor(config ResourceConfig) *ResourceMonitor {
	return &ResourceMonitor{
		config: config,
		stop:   make(chan bool),
		allocatedMemory: make([][]byte, 0),
	}
}

func (rm *ResourceMonitor) Start() {
	log.Printf("开始监控资源使用情况...")
	log.Printf("目标配置: 内存 %.1f%%, CPU %.1f%%, 磁盘 %.1f%%", 
		rm.config.MemoryPercent, rm.config.CPUPercent, rm.config.DiskPercent)

	ticker := time.NewTicker(rm.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.monitorAndAdjust()
		case <-rm.stop:
			log.Println("停止监控")
			// 调用完整的清理方法
			rm.cleanupAllResources()
			return
		}
	}
}

func (rm *ResourceMonitor) Stop() {
	// 发送停止信号，触发Start方法中的清理流程
	select {
	case <-rm.stop:
		// 已经停止
		return
	default:
		close(rm.stop)
	}
}

func (rm *ResourceMonitor) stopCPULoad() {
	rm.cpuLoadMutex.Lock()
	defer rm.cpuLoadMutex.Unlock()
	
	if rm.activeCPULoad && rm.cpuLoadStop != nil {
		log.Println("正在停止CPU负载...")
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
			log.Println("CPU负载已停止")
		case <-time.After(3 * time.Second):
			log.Println("CPU负载停止超时，强制停止")
		}
		
		rm.activeCPULoad = false
		rm.cpuLoadStop = make(chan bool)
		rm.cpuLoadWg = sync.WaitGroup{}
	}
}

func (rm *ResourceMonitor) monitorAndAdjust() {
	// 检查是否收到停止信号
	select {
	case <-rm.stop:
		return
	default:
	}
	
	// 获取当前资源使用情况
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("获取内存信息失败: %v", err)
		return
	}

	// 检查是否收到停止信号
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

	// 检查是否收到停止信号
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

	// 检查是否收到停止信号
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

	// 按优先级调整资源使用：先磁盘，再CPU，最后内存
	rm.adjustDiskUsage(currentDiskPercent, diskInfo)
	
	// 检查是否收到停止信号
	select {
	case <-rm.stop:
		return
	default:
	}
	
	// 等待一下让磁盘操作完成，避免临时内存占用影响
	time.Sleep(500 * time.Millisecond)
	
	// 检查是否收到停止信号
	select {
	case <-rm.stop:
		return
	default:
	}
	
	// 重新获取内存信息，因为磁盘操作可能影响了内存使用
	memInfoAfterDisk, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("重新获取内存信息失败: %v", err)
		return
	}
	
	// 检查是否收到停止信号
	select {
	case <-rm.stop:
		return
	default:
	}
	
	rm.adjustCPUUsage(currentCPUPercent)
	
	// 检查是否收到停止信号
	select {
	case <-rm.stop:
		return
	default:
	}
	
	rm.adjustMemoryUsage(memInfoAfterDisk.UsedPercent, memInfoAfterDisk)
}

func (rm *ResourceMonitor) adjustMemoryUsage(currentPercent float64, memInfo *mem.VirtualMemoryStat) {
	if currentPercent < rm.config.MemoryPercent-2 {
		// 内存使用率过低，增加内存使用
		targetBytes := uint64(float64(rm.config.MemoryPercent-currentPercent) / 100.0 * float64(memInfo.Total))
		rm.allocateMemory(targetBytes)
	} else if currentPercent > rm.config.MemoryPercent+2 {
		// 内存使用过高，自动释放部分内存
		rm.releaseMemory(currentPercent, memInfo)
	}
}

func (rm *ResourceMonitor) releaseMemory(currentPercent float64, memInfo *mem.VirtualMemoryStat) {
	rm.memoryMutex.Lock()
	defer rm.memoryMutex.Unlock()
	
	if len(rm.allocatedMemory) == 0 {
		return
	}
	
	// 计算需要释放的内存大小 - 更激进的释放策略
	excessPercent := currentPercent - rm.config.MemoryPercent
	
	// 根据超出程度调整释放策略
	var releaseRatio float64
	if excessPercent <= 5 {
		releaseRatio = 0.3 // 超出5%以内，释放30%
	} else if excessPercent <= 10 {
		releaseRatio = 0.5 // 超出5-10%，释放50%
	} else {
		releaseRatio = 0.7 // 超出10%以上，释放70%
	}
	
	log.Printf("内存使用率 %.1f%% 过高，目标 %.1f%%，超出 %.1f%%，释放 %.0f%% 分配的内存", 
		currentPercent, rm.config.MemoryPercent, excessPercent, releaseRatio*100)
	
	// 计算要释放的内存块数量
	blocksToRelease := int(float64(len(rm.allocatedMemory)) * releaseRatio)
	if blocksToRelease < 1 {
		blocksToRelease = 1 // 至少释放一个块
	}
	
	// 释放内存块（从最后分配的开始释放）
	releasedBytes := uint64(0)
	for i := 0; i < blocksToRelease && len(rm.allocatedMemory) > 0; i++ {
		lastIndex := len(rm.allocatedMemory) - 1
		releasedBytes += uint64(len(rm.allocatedMemory[lastIndex]))
		
		// 移除最后一个内存块
		rm.allocatedMemory = rm.allocatedMemory[:lastIndex]
	}
	
	log.Printf("已释放 %d 个内存块，总大小 %.2f MB", blocksToRelease, float64(releasedBytes)/1024/1024)
	
	// 强制垃圾回收
	runtime.GC()
	
	// 等待一下让GC完成
	time.Sleep(200 * time.Millisecond)
}

func (rm *ResourceMonitor) adjustCPUUsage(currentPercent float64) {
	rm.cpuLoadMutex.Lock()
	defer rm.cpuLoadMutex.Unlock()

	if currentPercent < rm.config.CPUPercent-5 {
		// 启动CPU负载
		if !rm.activeCPULoad {
			rm.startCPULoad()
		}
	} else if currentPercent > rm.config.CPUPercent+5 {
		// 停止CPU负载
		if rm.activeCPULoad {
			rm.stopCPULoad()
		}
	}
}

func (rm *ResourceMonitor) startCPULoad() {
	if rm.activeCPULoad {
		return
	}
	
	rm.activeCPULoad = true
	rm.cpuLoadWg.Add(1)
	
	go func() {
		defer rm.cpuLoadWg.Done()
		
		// 计算需要的CPU核心数，确保至少使用1个核心
		targetCores := int(float64(runtime.NumCPU()) * rm.config.CPUPercent / 100.0)
		if targetCores < 1 {
			targetCores = 1
		}
		
		log.Printf("启动CPU负载，使用 %d 个核心，目标CPU使用率 %.1f%%", targetCores, rm.config.CPUPercent)
		
		// 创建CPU负载goroutine
		for i := 0; i < targetCores; i++ {
			rm.cpuLoadWg.Add(1)
			go rm.cpuWorker(i)
		}
	}()
}

func (rm *ResourceMonitor) cpuWorker(id int) {
	defer rm.cpuLoadWg.Done()
	
	// 计算每个worker需要的工作时间比例
	workRatio := rm.config.CPUPercent / 100.0
	workTime := time.Duration(float64(10) * workRatio) // 10ms周期中的工作时间
	sleepTime := time.Duration(10) - workTime
	
	log.Printf("CPU Worker %d 启动: 工作时间 %.2fms, 睡眠时间 %.2fms", id, float64(workTime)/float64(time.Millisecond), float64(sleepTime)/float64(time.Millisecond))
	
	for {
		select {
		case <-rm.cpuLoadStop:
			return
		default:
			// 工作阶段：执行计算密集型操作
			start := time.Now()
			for time.Since(start) < workTime {
				rm.doCPUWork()
			}
			
			// 睡眠阶段：让出CPU
			if sleepTime > 0 {
				time.Sleep(sleepTime)
			}
		}
	}
}

func (rm *ResourceMonitor) doCPUWork() {
	// 执行计算密集型操作来产生CPU负载
	var result float64
	for i := 0; i < 50000; i++ {
		result += float64(i) * float64(i) * float64(i)
	}
	_ = result // 避免编译器优化
}

func (rm *ResourceMonitor) adjustDiskUsage(currentPercent float64, diskInfo *disk.UsageStat) {
	// 检查是否收到停止信号
	select {
	case <-rm.stop:
		return
	default:
	}
	
	if currentPercent < rm.config.DiskPercent {
		// 计算需要增加的磁盘空间
		targetBytes := uint64(float64(rm.config.DiskPercent-currentPercent) / 100.0 * float64(diskInfo.Total))
		rm.createTempFiles(targetBytes)
	} else if currentPercent > rm.config.DiskPercent+5 {
		// 磁盘使用过高，清理临时文件
		rm.cleanupTempFiles()
	}
}

func (rm *ResourceMonitor) allocateMemory(bytes uint64) {
	if bytes == 0 {
		return // 不分配0字节内存
	}
	
	rm.memoryMutex.Lock()
	defer rm.memoryMutex.Unlock()
	
	// 分配内存
	data := make([]byte, bytes)
	for i := range data {
		data[i] = byte(i % 256)
	}
	
	// 记录分配的内存
	rm.allocatedMemory = append(rm.allocatedMemory, data)
	
	log.Printf("已分配 %.2f MB 内存", float64(bytes)/1024/1024)
}

func (rm *ResourceMonitor) createTempFiles(targetBytes uint64) {
	rm.diskMutex.Lock()
	defer rm.diskMutex.Unlock()
	
	// 计算需要创建的文件数量（每个文件5G，减少内存占用）
	fileSize := uint64(5 * 1024 * 1024 * 1024) // 5G
	numFiles := targetBytes / fileSize + 1
	
	log.Printf("需要创建 %d 个文件，总大小 %.2f MB", numFiles, float64(numFiles*fileSize)/1024/1024)
	
	// 创建文件
	for i := 0; i < int(numFiles); i++ {
		// 检查是否收到停止信号
		select {
		case <-rm.stop:
			log.Println("收到停止信号，停止创建文件")
			return
		default:
		}
		
		filename := fmt.Sprintf("/tmp/occupy_%d_%d.tmp", time.Now().Unix(), i)
		file, err := os.Create(filename)
		if err != nil {
			log.Printf("创建临时文件失败: %v", err)
			continue
		}
		
		// 分块写入数据，减少内存占用
		chunkSize := uint64(10 * 1024 * 1024) // 10MB chunks
		for written := uint64(0); written < fileSize; written += chunkSize {
			// 更频繁地检查停止信号
			select {
			case <-rm.stop:
				log.Println("收到停止信号，停止写入文件")
				file.Close()
				// 删除未完成的文件
				os.Remove(filename)
				return
			default:
			}
			
			remaining := fileSize - written
			if remaining < chunkSize {
				chunkSize = remaining
			}
			
			data := make([]byte, chunkSize)
			for j := range data {
				data[j] = byte((j + int(written) + i) % 256)
			}
			
			_, err = file.Write(data)
			if err != nil {
				log.Printf("写入临时文件失败: %v", err)
				break
			}
			
			// 释放临时数据内存
			data = nil
			
			// 每写入一个块后检查停止信号
			select {
			case <-rm.stop:
				log.Println("收到停止信号，停止写入文件")
				file.Close()
				// 删除未完成的文件
				os.Remove(filename)
				return
			default:
			}
		}
		
		file.Close()
		
		log.Printf("已创建临时文件: %s (%.2f MB)", filename, float64(fileSize)/1024/1024)
		
		// 每个文件创建完成后检查停止信号
		select {
		case <-rm.stop:
			log.Println("收到停止信号，停止创建文件")
			return
		default:
		}
	}
}

func (rm *ResourceMonitor) cleanupTempFiles() {
	rm.diskMutex.Lock()
	defer rm.diskMutex.Unlock()
	
	// 使用shell命令删除所有临时文件
	cmd := exec.Command("sh", "-c", "rm -f /tmp/occupy_*.tmp")
	err := cmd.Run()
	if err != nil {
		log.Printf("清理临时文件失败: %v", err)
	} else {
		log.Println("临时文件清理完成")
	}
}

func (rm *ResourceMonitor) cleanupMemory() {
	rm.memoryMutex.Lock()
	defer rm.memoryMutex.Unlock()
	
	if len(rm.allocatedMemory) > 0 {
		log.Printf("清理 %d 块内存分配，总大小 %.2f MB", 
			len(rm.allocatedMemory), 
			float64(rm.getTotalAllocatedMemory())/1024/1024)
		
		// 清空内存数组，让GC回收内存
		rm.allocatedMemory = make([][]byte, 0)
		
		// 强制垃圾回收
		runtime.GC()
		log.Println("内存清理完成")
	}
}

func (rm *ResourceMonitor) getTotalAllocatedMemory() uint64 {
	var total uint64
	for _, mem := range rm.allocatedMemory {
		total += uint64(len(mem))
	}
	return total
}

func (rm *ResourceMonitor) cleanupAllTempFiles() {
	rm.diskMutex.Lock()
	defer rm.diskMutex.Unlock()
	
	// 使用shell命令删除所有临时文件
	cmd := exec.Command("sh", "-c", "rm -f /tmp/occupy_*.tmp")
	err := cmd.Run()
	if err != nil {
		log.Printf("清理临时文件失败: %v", err)
	} else {
		log.Println("磁盘文件清理完成")
	}
}

func (rm *ResourceMonitor) cleanupAllResources() {
	log.Println("开始清理所有资源...")
	
	// 按顺序清理资源：CPU -> 内存 -> 磁盘
	log.Println("停止CPU负载")
	rm.stopCPULoad()
	
	log.Println("清理内存负载")
	rm.cleanupMemory()
	
	log.Println("清理磁盘文件")
	rm.cleanupAllTempFiles()
	
	log.Println("所有资源清理完成")
}

func main() {
	var memoryPercent, cpuPercent, diskPercent float64
	var interval time.Duration

	rootCmd := &cobra.Command{
		Use:   "go-occupy",
		Short: "固定比例占比内存、CPU、磁盘空间",
		Long: `一个用于监控和调整系统资源使用比例的工具。
可以设置目标的内存、CPU和磁盘使用百分比，程序会自动调整以达到目标比例。`,
		Run: func(cmd *cobra.Command, args []string) {
			config := ResourceConfig{
				MemoryPercent: memoryPercent,
				CPUPercent:    cpuPercent,
				DiskPercent:   diskPercent,
				Interval:      interval,
			}

			monitor := NewResourceMonitor(config)

			// 设置信号处理
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				<-sigChan
				log.Println("收到退出信号，正在停止...")
				monitor.Stop()
			}()

			monitor.Start()
		},
	}

	rootCmd.Flags().Float64VarP(&memoryPercent, "memory", "m", 50.0, "目标内存使用百分比")
	rootCmd.Flags().Float64VarP(&cpuPercent, "cpu", "c", 30.0, "目标CPU使用百分比")
	rootCmd.Flags().Float64VarP(&diskPercent, "disk", "d", 40.0, "目标磁盘使用百分比")
	rootCmd.Flags().DurationVarP(&interval, "interval", "i", 5*time.Second, "监控间隔")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "执行命令失败: %v\n", err)
		os.Exit(1)
	}
} 