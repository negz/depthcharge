package managed

import (
	"fmt"
	"github.com/crossplane/conformance/internal/providerScale/cmd/common"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func RunExperiment(mrTemplatePaths map[string]int, clean bool) ([]*common.Result, error) {
	var timeToReadinessResults []*common.Result

	for mrPath, count := range mrTemplatePaths {
		for i := 1; i <= count; i++ {
			cmd, err := exec.Command("./internal/providerScale/cmd/managed/manage-mr.sh", "apply", mrPath, strconv.Itoa(i)).Output()
			fmt.Print(string(cmd))
			if err != nil {
				return nil, err
			}
		}
	}
	for mrPath, count := range mrTemplatePaths {
		if err := checkReadiness(mrPath, count); err != nil {
			return nil, err
		}
	}
	for mrPath, count := range mrTemplatePaths {
		timeToReadinessResult, err := calculateReadinessDuration(mrPath, count)
		if err != nil {
			return nil, err
		}
		timeToReadinessResults = append(timeToReadinessResults, timeToReadinessResult)
	}
	if clean {
		for mrPath, count := range mrTemplatePaths {
			fmt.Println("Deleting resources...")
			for i := 1; i <= count; i++ {
				cmd, err := exec.Command("./internal/providerScale/cmd/managed/manage-mr.sh", "delete", mrPath, strconv.Itoa(i)).Output()
				fmt.Print(string(cmd))
				if err != nil {
					return nil, err
				}
			}
		}
		for mrPath, count := range mrTemplatePaths {
			i := 1
			for i <= count {
				fmt.Println("Checking deletion of resources...")
				cmd, err := exec.Command("./internal/providerScale/cmd/managed/checkDeletion.sh", mrPath, strconv.Itoa(i)).Output()
				if err != nil {
					return nil, err
				}
				if string(cmd) != "" {
					time.Sleep(10 * time.Second)
					continue
				}
				i++
			}
		}
	}
	return timeToReadinessResults, nil
}

func checkReadiness(mrPath string, count int) error {
	i := 1
	for i <= count {
		fmt.Println("Checking readiness of resources...")
		isReady, _ := exec.Command("./internal/providerScale/cmd/managed/checkReadiness.sh", mrPath, strconv.Itoa(i)).Output()
		if !strings.Contains(string(isReady), "True") {
			time.Sleep(10 * time.Second)
			continue
		}
		i++
	}

	return nil
}

func calculateReadinessDuration(mrPath string, count int) (*common.Result, error) {
	result := &common.Result{}
	for i := 1; i <= count; i++ {
		fmt.Println("Calculating readiness durations of resources...")
		creationTimeByte, err := exec.Command("./internal/providerScale/cmd/managed/getCreationTime.sh", mrPath, strconv.Itoa(i)).Output()
		if err != nil {
			return nil, err
		}
		readinessTimeByte, err := exec.Command("./internal/providerScale/cmd/managed/getReadinessTime.sh", mrPath, strconv.Itoa(i)).Output()
		if err != nil {
			return nil, err
		}
		creationTimeString := string(creationTimeByte)
		creationTimeString = creationTimeString[:strings.Index(creationTimeString, `Z`)+1]
		creationTime, err := time.Parse(time.RFC3339, creationTimeString)
		if err != nil {
			return nil, err
		}
		readinessTimeString := string(readinessTimeByte)
		readinessTimeString = readinessTimeString[strings.Index(readinessTimeString, `"`)+1 : strings.Index(readinessTimeString, `Z`)+1]
		readinessTime, err := time.Parse(time.RFC3339, readinessTimeString)
		if err != nil {
			return nil, err
		}
		diff := readinessTime.Sub(creationTime)
		result.Data = append(result.Data, common.Data{Value: diff.Seconds()})
	}
	result.Metric = mrPath
	result.Average, result.Peak = common.CalculateAverageAndPeak(result.Data)
	return result, nil
}
