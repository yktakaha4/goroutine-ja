package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/trace"
	"sync"
	"time"
)

type (
	Bean       int
	GroundBean int
	Water      int
	HotWater   int
	Coffee     int
)

const (
	GramBeans          Bean       = 1
	GramGroundBeans    GroundBean = 1
	MilliLiterWater    Water      = 1
	MilliLiterHotWater HotWater   = 1
	CupsCoffee         Coffee     = 1
)

func (w Water) String() string {
	return fmt.Sprintf("%d[ml] water", int(w))
}

func (hw HotWater) String() string {
	return fmt.Sprintf("%d[ml] hot water", int(hw))
}

func (b Bean) String() string {
	return fmt.Sprintf("%d[g] beans", int(b))
}

func (gb GroundBean) String() string {
	return fmt.Sprintf("%d[g] ground beans", int(gb))
}

func (cups Coffee) String() string {
	return fmt.Sprintf("%d cup(s) coffee", int(cups))
}

// 1カップのコーヒーを淹れるのに必要な水の量
func (cups Coffee) Water() Water {
	return Water(180*cups) / MilliLiterWater
}

// 1カップのコーヒーを淹れるのに必要なお湯の量
func (cups Coffee) HotWater() HotWater {
	return HotWater(180*cups) / MilliLiterHotWater
}

// 1カップのコーヒーを淹れるのに必要な豆の量
func (cups Coffee) Beans() Bean {
	return Bean(20*cups) / GramBeans
}

// 1カップのコーヒーを淹れるのに必要な粉の量
func (cups Coffee) GroundBeans() GroundBean {
	return GroundBean(20*cups) / GramGroundBeans
}

func main() {
	f, err := os.Create("trace.out")
	if err != nil {
		log.Fatalln("Error:", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalln("Error:", err)
		}
	}()

	if err := trace.Start(f); err != nil {
		log.Fatalln("Error:", err)
	}
	defer trace.Stop()

	_main()
}

func _main() {
	// 作るコーヒーの数
	const amountCoffee = 20 * CupsCoffee

	ctx, task := trace.NewTask(context.Background(), "make coffe")
	defer task.End()

	// 材料
	water := amountCoffee.Water()
	beans := amountCoffee.Beans()

	fmt.Println(water)
	fmt.Println(beans)

	var wg sync.WaitGroup

	// お湯を沸かす
	var hotWater HotWater
	var hwmu sync.Mutex
	for water > 0 {
		water -= 600 * MilliLiterWater
		// wgに1を加える
		wg.Add(1)
		go func() {
			// deferでwg.Doneを仕掛ける
			defer wg.Done()
			hw := boil(ctx, 600*MilliLiterWater)
			hwmu.Lock()
			defer hwmu.Unlock()
			hotWater += hw
		}()
	}

	// 豆を挽く
	var groundBeans GroundBean
	var gbmu sync.Mutex
	for beans > 0 {
		beans -= 20 * GramBeans
		wg.Add(1)
		go func() {
			defer wg.Done()
			gb := grind(ctx, 20*GramBeans)
			// gbmuでロックを取る
			gbmu.Lock()
			// gbmuのロックをdeferで解除する
			defer gbmu.Unlock()
			groundBeans += gb
		}()
	}

	wg.Wait()
	fmt.Println(hotWater)
	fmt.Println(groundBeans)

	// コーヒーを淹れる
	var wg2 sync.WaitGroup
	var coffee Coffee
	var cfmu sync.Mutex
	cups := 4 * CupsCoffee
	for hotWater >= cups.HotWater() && groundBeans >= cups.GroundBeans() {
		hotWater -= cups.HotWater()
		groundBeans -= cups.GroundBeans()
		// wg2に1加える
		wg2.Add(1)
		go func() {
			// wg2のDoneをdeferで呼ぶ
			defer wg2.Done()
			cf := brew(ctx, cups.HotWater(), cups.GroundBeans())
			cfmu.Lock()
			defer cfmu.Unlock()
			coffee += cf
		}()
	}

	// wg2を使って待ち合わせ
	wg2.Wait()
	fmt.Println(coffee)
}

// お湯を沸かす
func boil(ctx context.Context, water Water) HotWater {
	defer trace.StartRegion(ctx, "boil").End()
	time.Sleep(400 * time.Millisecond)
	return HotWater(water)
}

// コーヒー豆を挽く
func grind(ctx context.Context, beans Bean) GroundBean {
	defer trace.StartRegion(ctx, "grind").End()
	time.Sleep(200 * time.Millisecond)
	return GroundBean(beans)
}

// コーヒーを淹れる
func brew(ctx context.Context, hotWater HotWater, groundBeans GroundBean) Coffee {
	defer trace.StartRegion(ctx, "brew").End()
	time.Sleep(1 * time.Second)
	// 少ない方を優先する
	cups1 := Coffee(hotWater / (1 * CupsCoffee).HotWater())
	cups2 := Coffee(groundBeans / (1 * CupsCoffee).GroundBeans())
	if cups1 < cups2 {
		return cups1
	}
	return cups2
}
