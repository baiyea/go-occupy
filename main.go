package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go-occupy/pkg/occupy"
)

var (
	memoryPercent float64
	cpuPercent    float64
	diskPercent   float64
	interval      time.Duration
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "go-occupy",
		Short: "Go-Occupy 是一个系统资源占用工具",
		Long: `Go-Occupy 是一个用于测试和演示系统资源占用的工具。
它可以模拟占用内存、CPU和磁盘空间，用于系统压力测试和性能评估。`,
		Run: runOccupy,
	}

	// 添加命令行参数
	rootCmd.Flags().Float64VarP(&memoryPercent, "memory", "m", 50.0, "目标内存使用百分比 (0-100)")
	rootCmd.Flags().Float64VarP(&cpuPercent, "cpu", "c", 30.0, "目标CPU使用百分比 (0-100)")
	rootCmd.Flags().Float64VarP(&diskPercent, "disk", "d", 40.0, "目标磁盘使用百分比 (0-100)")
	rootCmd.Flags().DurationVarP(&interval, "interval", "i", 5*time.Second, "监控间隔时间")

	// 添加子命令
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(helpCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runOccupy(cmd *cobra.Command, args []string) {
	// 验证参数
	if memoryPercent < 0 || memoryPercent > 100 {
		log.Fatal("内存百分比必须在 0-100 之间")
	}
	if cpuPercent < 0 || cpuPercent > 100 {
		log.Fatal("CPU百分比必须在 0-100 之间")
	}
	if diskPercent < 0 || diskPercent > 100 {
		log.Fatal("磁盘百分比必须在 0-100 之间")
	}

	// 创建资源配置
	config := occupy.ResourceConfig{
		MemoryPercent: memoryPercent,
		CPUPercent:    cpuPercent,
		DiskPercent:   diskPercent,
		Interval:      interval,
	}

	// 创建资源监控器
	monitor := occupy.NewResourceMonitor(config)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动监控
	go monitor.Start()

	// 等待信号
	<-sigChan
	log.Println("收到停止信号，正在优雅关闭...")

	// 停止监控（会等待清理完成）
	monitor.Stop()
	
	log.Println("程序已退出")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Go-Occupy v1.0.0")
		fmt.Println("一个系统资源占用工具")
	},
}

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "显示帮助信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Go-Occupy 使用说明:")
		fmt.Println("")
		fmt.Println("基本用法:")
		fmt.Println("  go-occupy                    # 使用默认配置")
		fmt.Println("  go-occupy -m 80 -c 70 -d 90  # 自定义配置")
		fmt.Println("")
		fmt.Println("参数说明:")
		fmt.Println("  -m, --memory   目标内存使用百分比 (默认: 50)")
		fmt.Println("  -c, --cpu      目标CPU使用百分比 (默认: 30)")
		fmt.Println("  -d, --disk     目标磁盘使用百分比 (默认: 40)")
		fmt.Println("  -i, --interval 监控间隔时间 (默认: 5s)")
		fmt.Println("")
		fmt.Println("示例:")
		fmt.Println("  go-occupy -m 30 -c 20 -d 30  # 开发模式")
		fmt.Println("  go-occupy -m 50 -c 30 -d 40  # 测试模式")
		fmt.Println("  go-occupy -m 80 -c 70 -d 60  # 高负载模式")
		fmt.Println("")
		fmt.Println("控制:")
		fmt.Println("  按 Ctrl+C 停止程序")
	},
} 