package measure

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

var wg sync.WaitGroup
var l sync.Mutex

// Measure is test lib for stress test
type Measure struct {
	current  int
	result   map[int](map[int]time.Duration)
	stage    map[int]string
	flag     map[int](map[int]int)
	callback map[int](func(int) int)
}

// NewMeasure return a Measure inst
func NewMeasure() *Measure {
	return &Measure{
		current:  0,
		result:   make(map[int](map[int]time.Duration)),
		stage:    make(map[int]string),
		flag:     make(map[int](map[int]int)),
		callback: make(map[int](func(int) int)),
	}
}

// Stage define a stage
func (measure *Measure) Stage(name string, callback func(int) int) {
	measure.stage[measure.current] = name
	measure.callback[measure.current] = callback
	measure.current++
}

// Run test
func (measure *Measure) Run(concurrency, total int) {
	nextRun := 0
	totalStage := len(measure.stage)
	for i := 0; i < total; i++ {
		measure.result[i] = make(map[int]time.Duration)
		measure.flag[i] = make(map[int]int)
	}
	var exec func(int)
	exec = func(runIndex int) {
		// fmt.Printf("exec - %d\n", runIndex)
		defer wg.Done()
		for stageIndex := 0; stageIndex < totalStage; stageIndex++ {
			start := time.Now()
			flag := measure.callback[stageIndex](runIndex)
			elapsed := time.Since(start)
			measure.flag[runIndex][stageIndex] = flag
			measure.result[runIndex][stageIndex] = elapsed
		}
		l.Lock()
		if nextRun != total {
			wg.Add(1)
			go exec(nextRun)
			nextRun++
		}
		l.Unlock()
		// fmt.Printf("exec done - %d\n", runIndex)
	}
	wg.Add(concurrency)
	for ; nextRun < concurrency; nextRun++ {
		go exec(nextRun)
	}
	wg.Wait()
}

func (measure *Measure) print(avg, pct90, pct95, pct99 time.Duration) {
	fmt.Printf("average: %s\n", avg.String())
	fmt.Printf("pct90: %s\n", pct90.String())
	fmt.Printf("pct95: %s\n", pct95.String())
	fmt.Printf("pct99: %s\n\n", pct99.String())
}

// Print show detail
func (measure *Measure) Print(filter []int, showTotal bool) {
	shadowMap := make(map[int](map[int]time.Duration))
	for k, v := range measure.result {
		shadowMap[k] = v
	}
	stageLength := len(measure.stage)
	validCount := 0
	for runIndex := range shadowMap {
		isValid := true
		for i := 0; i < stageLength; i++ {
			flag := measure.flag[runIndex][i]
			if filter[i]&flag == 0 {
				isValid = false
				break
			}
		}
		if !isValid {
			delete(shadowMap, runIndex)
		} else {
			validCount++
		}
	}
	pct90 := validCount * 90 / 100
	pct95 := validCount * 95 / 100
	pct99 := validCount * 99 / 100
	if pct99 >= validCount {
		pct99 = validCount - 1
	}
	// print cost for every stage
	result := make([]int, validCount)
	var validIndex int
	for stageIndex := 0; stageIndex < stageLength; stageIndex++ {
		validIndex = 0
		for runIndex := range shadowMap {
			result[validIndex] = int(shadowMap[runIndex][stageIndex])
			validIndex++
		}
		sort.Ints(result)
		fmt.Printf("stage - %s\n-----\n", measure.stage[stageIndex])
		var total int64
		for _, value := range result {
			total += int64(value)
		}
		avg := total / int64(validCount)
		measure.print(time.Duration(avg), time.Duration(result[pct90]), time.Duration(result[pct95]), time.Duration(result[pct99]))
	}
	// print total usage
	if showTotal {
		totalCost := make([]int, validCount)
		for i := 0; i < stageLength; i++ {
			totalCost[i] = 0
		}
		validIndex = 0
		for _, runResult := range shadowMap {
			for stageIndex := 0; stageIndex < stageLength; stageIndex++ {
				totalCost[validIndex] += int(runResult[stageIndex])
			}
			validIndex++
		}
		sort.Ints(totalCost)
		var total int64
		for _, value := range totalCost {
			total += int64(value)
		}
		fmt.Printf("total\n-----\n")
		avg := total / int64(validCount)
		measure.print(time.Duration(avg), time.Duration(result[pct90]), time.Duration(result[pct95]), time.Duration(result[pct99]))
	}
	fmt.Printf("%d valid result\n", validCount)
}

// GetResult return result
func (measure *Measure) GetResult() map[int](map[int]time.Duration) {
	return measure.result
}
