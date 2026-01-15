package main

import (
    "fmt"
    "math/rand"
    "sort"
    "strings"
    "sync"
    "time"

    "github.com/fatih/color"
)

// DNSServer 表示DNS服务器信息
type DNSServer struct {
    Name   string
    IP     string
    Region string
}

// TestResult 表示测试结果
type TestResult struct {
    Server          DNSServer
    AvgResponseTime float64
    MinResponseTime float64
    MaxResponseTime float64
    ResponseTimes   []float64
    Status          string
    SuccessCount    int
    Connectivity    bool
    FirstIP         string
}

// DNSTester DNS测试器
type DNSTester struct {
    DNSServers  []DNSServer
    TestDomains []string
}

// LoadCustomDNS 加载自定义DNS服务器列表
func (dt *DNSTester) LoadCustomDNS() {
    defaultDNS := []DNSServer{
        {Name: "Google DNS", IP: "8.8.8.8", Region: "Global"},
        {Name: "Google DNS", IP: "8.8.4.4", Region: "Global"},
        {Name: "Cloudflare DNS", IP: "1.1.1.1", Region: "Global"},
        {Name: "Cloudflare DNS", IP: "1.0.0.1", Region: "Global"},
        {Name: "OpenDNS", IP: "208.67.222.222", Region: "Global"},
        {Name: "Quad9", IP: "9.9.9.9", Region: "Global"},
        {Name: "阿里DNS", IP: "223.5.5.5", Region: "China"},
        {Name: "阿里DNS", IP: "223.6.6.6", Region: "China"},
        {Name: "腾讯DNS", IP: "119.29.29.29", Region: "China"},
        {Name: "114 DNS", IP: "114.114.114.114", Region: "China"},
        {Name: "百度DNS", IP: "180.76.76.76", Region: "China"},
        {Name: "CNNIC DNS", IP: "1.2.4.8", Region: "China"},
    }

    dt.DNSServers = defaultDNS
    dt.TestDomains = []string{"www.google.com", "www.baidu.com", "www.qq.com", "www.taobao.com"}
}

// MockTestDNSResponseTime 模拟DNS测试
func (dt *DNSTester) MockTestDNSResponseTime(server DNSServer, domain string) TestResult {
    // 模拟DNS查询延迟
    times := make([]float64, 0, 3)

    for i := 0; i < 3; i++ {
        // 生成随机响应时间（模拟真实情况）
        baseTime := rand.Float64() * 200 // 0-200ms
        jitter := rand.Float64() * 20   // 额外抖动
        responseTime := baseTime + jitter
        times = append(times, responseTime)
    }

    // 计算统计数据
    var sum float64
    min := times[0]
    max := times[0]
    for _, t := range times {
        sum += t
        if t < min {
            min = t
        }
        if t > max {
            max = t
        }
    }
    avg := sum / float64(len(times))

    // 模拟连通性测试
    connectivity := rand.Intn(2) == 1 // 50% 概率连通

    return TestResult{
        Server:          server,
        AvgResponseTime: avg,
        MinResponseTime: min,
        MaxResponseTime: max,
        ResponseTimes:   times,
        Status:          "Success",
        SuccessCount:    len(times),
        Connectivity:    connectivity,
        FirstIP:         fmt.Sprintf("192.168.%d.%d", rand.Intn(255), rand.Intn(255)),
    }
}

// TestAllDNS 测试所有DNS服务器
func (dt *DNSTester) TestAllDNS(concurrency int) []TestResult {
    results := make([]TestResult, 0, len(dt.DNSServers))
    var mu sync.Mutex
    var wg sync.WaitGroup

    semaphore := make(chan struct{}, concurrency)

    for _, server := range dt.DNSServers {
        wg.Add(1)
        semaphore <- struct{}{} // 获取信号量

        go func(s DNSServer) {
            defer wg.Done()
            defer func() { <-semaphore }() // 释放信号量

            result := dt.MockTestDNSResponseTime(s, dt.TestDomains[0])

            mu.Lock()
            results = append(results, result)
            mu.Unlock()

            // 打印进度
            fmt.Print(".")
        }(server)
    }

    wg.Wait()
    close(semaphore)

    // 按平均响应时间排序
    sort.Slice(results, func(i, j int) bool {
        if results[i].AvgResponseTime == float64(-1) {
            return false
        }
        if results[j].AvgResponseTime == float64(-1) {
            return true
        }
        return results[i].AvgResponseTime < results[j].AvgResponseTime
    })

    return results
}

// DisplayResults 显示测试结果
func (dt *DNSTester) DisplayResults(results []TestResult) {
    fmt.Println()
    fmt.Println("DNS服务器速度测试结果（云端模拟版）")
    fmt.Println("=" + strings.Repeat("=", 100))

    headerFormat := "%-4s %-20s %-15s %-8s %-15s %-15s %-10s %-10s %-10s\n"
    fmt.Printf(headerFormat,
        "排名", "DNS名称", "IP地址", "地区", "平均响应时间", "最小/最大", "成功率", "连通性", "状态")

    fmt.Println(strings.Repeat("-", 100))

    for i, result := range results {
        rank := i + 1

        var avgTimeStr, minMaxStr, successRate, connectivity, status string
        if result.AvgResponseTime == float64(-1) {
            avgTimeStr = "超时/错误"
            minMaxStr = "-/-"
            successRate = "0%"
            connectivity = "❌"
            status = result.Status
        } else {
            avgTimeStr = fmt.Sprintf("%.2fms", result.AvgResponseTime)
            minMaxStr = fmt.Sprintf("%.2f/%.2f", result.MinResponseTime, result.MaxResponseTime)
            successRate = fmt.Sprintf("%d%%", result.SuccessCount*100/3)
            if result.Connectivity {
                connectivity = "✅"
            } else {
                connectivity = "❌"
            }
            status = "正常"
        }

        region := result.Server.Region
        if region == "" {
            region = "Unknown"
        }

        fmt.Printf(headerFormat,
            fmt.Sprintf("%d", rank),
            result.Server.Name,
            result.Server.IP,
            region,
            avgTimeStr,
            minMaxStr,
            successRate,
            connectivity,
            status)
    }
}

// GetBestDNS 获取最快的几个DNS服务器
func (dt *DNSTester) GetBestDNS(results []TestResult, count int) []TestResult {
    validResults := make([]TestResult, 0)
    for _, r := range results {
        if r.AvgResponseTime != float64(-1) && r.SuccessCount > 0 {
            validResults = append(validResults, r)
        }
    }

    if len(validResults) < count {
        return validResults
    }
    return validResults[:count]
}

func main() {
    rand.Seed(time.Now().UnixNano())
    
    color.Cyan("DNS服务器速度测试工具 (云端模拟版)")
    fmt.Println(strings.Repeat("=", 60))

    tester := &DNSTester{}
    tester.LoadCustomDNS()

    fmt.Print("开始测试DNS服务器响应时间（模拟）...")
    results := tester.TestAllDNS(10) // 并发数为10
    fmt.Println(" 完成!")

    tester.DisplayResults(results)

    bestDNS := tester.GetBestDNS(results, 3)
    if len(bestDNS) > 0 {
        fmt.Println()
        fmt.Println("推荐的最快DNS服务器:")
        for i, dns := range bestDNS {
            connectivityStatus := "异常"
            if dns.Connectivity {
                connectivityStatus = "正常"
            }
            fmt.Printf("%d. %s (%s) - %.2fms (成功率: %d%%, 连通性: %s)\n",
                i+1, dns.Server.Name, dns.Server.IP,
                dns.AvgResponseTime,
                dns.SuccessCount*100/3,
                connectivityStatus)
        }
    } else {
        fmt.Println("没有找到可用的DNS服务器")
    }

    fmt.Println("\n注意：此版本为云端模拟版，实际DNS查询功能受限于云端环境限制。")
    fmt.Println("如需完整功能，请在本地环境安装Go后运行完整版本。")
}
