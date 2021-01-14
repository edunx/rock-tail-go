package tail

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSwitch(t *testing.T) {
	a := 1
	b := 2
	a = b
	fmt.Print(a)

}

func TestMap(t *testing.T) {
	a := make(map[string]int)

	a = nil
	_, ok := a["test"]
	if !ok {
		fmt.Println(111)
	}

}

func TestTail_Update(t *testing.T) {
	rand.Seed(time.Now().Unix())
	for {
		os.Remove("/home/suncle/goProject/src/rock/agent/resource/logs/error.log")
		t := rand.Intn(10) + 10
		time.Sleep(time.Duration(t) * time.Second)
	}
}

func TestTail_Update2(t *testing.T) {

	//for min := 10; min < 30; min++ {
	//	name := fmt.Sprintf("/home/suncle/goProject/src/rock/agent/resource/logs/error.2020-12-17.20.%d.log", min)
	//	f, _ := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	//	_ = f.Close()
	//}

	go func() {
		for {
			nowTime := time.Now().Format("2006-01-02.15")
			name := fmt.Sprintf("/home/suncle/goProject/src/rock/agent/resource/logs/error.%s.log", nowTime)
			//if _, err := os.Stat(name); os.IsNotExist(err) {
			//	time.Sleep(1 * time.Second)
			//	continue
			//}
			f, _ := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
			data := time.Now().Format("2006-01-02.15")
			_, _ = f.WriteString(fmt.Sprintf("%s\n", data))
			time.Sleep(1 * time.Second)
		}
	}()

	rand.Seed(time.Now().Unix())
	for {
		nowTime := time.Now().Format("2006-01-02.15")
		name := fmt.Sprintf("/home/suncle/goProject/src/rock/agent/resource/logs/error.%s.log", nowTime)
		_ = os.Remove(name)
		t := rand.Intn(10) + 10
		time.Sleep(time.Duration(t) * time.Second)
	}
}

func TestTail_DirWatcher(t *testing.T) {
	str := filepath.Dir("/home/suncle/goProject/src/rock/agent/resource/logs")
	t.Log(str)
}
