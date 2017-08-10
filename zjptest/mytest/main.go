package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"time"

	_ "github.com/Go-SQL-Driver/MySQL"
	"github.com/prometheus/alertmanager/provider/mem"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
)

//type LabelSet map[string]string

func main() {
	db, err := sql.Open("mysql", "root:101314@tcp(localhost:3306)/alertmanager2?charset=utf8")
	checkErr(err)
	db.Query("TRUNCATE TABLE alertmanager2.alerts")
	dir, err := ioutil.TempDir("", "alerts_test")
	if err != nil {
		fmt.Println("TempDir wrong")
	}
	defer os.RemoveAll(dir)

	marker := types.NewMarker()
	alerts, err := mem.NewAlerts(marker, 30*time.Minute, dir)
	if err != nil {
		fmt.Println("marker, alerts wrong")
	}

	//数据样例
	var (
		t0 = time.Now()
		t1 = t0.Add(150 * time.Minute)
	)
	insert := []*types.Alert{
		{
			Alert: model.Alert{
				Labels:       model.LabelSet{"bar": "foo"},
				Annotations:  model.LabelSet{"foo": "bar"},
				StartsAt:     t0,
				EndsAt:       t1,
				GeneratorURL: "http://example.com/prometheus",
			},
			UpdatedAt: t0,
			Timeout:   false,
		}, {
			Alert: model.Alert{
				Labels:       model.LabelSet{"bar": "foo2"},
				Annotations:  model.LabelSet{"foo": "bar2"},
				StartsAt:     t0,
				EndsAt:       t1,
				GeneratorURL: "http://example.com/prometheus",
			},
			UpdatedAt: t0,
			Timeout:   false,
		}, {
			Alert: model.Alert{
				Labels:       model.LabelSet{"bar": "foo3"},
				Annotations:  model.LabelSet{"foo": "bar3"},
				StartsAt:     t0,
				EndsAt:       t1,
				GeneratorURL: "http://example.com/prometheus",
			},
			UpdatedAt: t0,
			Timeout:   false,
		},
	}

	//插入数据到数据库中
	putmysql(db, insert)

	//从数据库中获取数据
	insert = getmysql(db)

	if err := alerts.Put(insert...); err != nil {
		fmt.Print("insert error:")
		fmt.Println(err)
	}

	for _, a := range insert {
		res, err := alerts.Get(a.Fingerprint())
		// fmt.Println("*********")
		fmt.Println(res.StartsAt)
		// fmt.Println(a)s
		// fmt.Println("*********")
		if err != nil {
			fmt.Println("retrieval error: %s", err)
		}
		if !alertsEqual(res, a) {
			fmt.Println("Unexpected alert: i")
			//t.Fatalf(pretty.Compare(res, a))
		}
	}

	// for _, a := range insert {
	// 	marker.SetActive(a.Fingerprint())
	// 	if !marker.Active(a.Fingerprint()) {
	// 		fmt.Println("error setting status: %v", a)
	// 		return
	// 	}
	// 	//fmt.Println(a.Labels)
	// }
	// time.Sleep(300 * time.Millisecond)
	// for i, a := range insert {
	// 	b, err := alerts.Get(a.Fingerprint())
	// 	fmt.Println(i)
	// 	fmt.Println(b)
	// 	fmt.Println(err)
	// 	fmt.Println("****testzjp main.go****")
	// 	//t.Fatalf(b.GeneratorURL)
	// 	if err != provider.ErrNotFound {
	// 		fmt.Println(err)
	// 		fmt.Println("alert %d didn't get GC'd", i)
	// 		return
	// 	}

	// 	s := marker.Status(a.Fingerprint())
	// 	if s.State != types.AlertStateUnprocessed {
	// 		fmt.Println("marker %d didn't get GC'd: %v", i, s)
	// 		return
	// 	}
	// }
	fmt.Println("xixixi")
	db.Close()
}

func putmysql(db *sql.DB, insert []*types.Alert) {
	//将数据装入数据库中
	for i, a := range insert {
		fmt.Println("alert :", i)
		fmt.Println(a.Alert)
		//插入到alerts表中
		labeltmp, err := json.Marshal(a.Labels)
		//fmt.Println(a.Labels)
		annotationtmp, err := json.Marshal(a.Annotations)
		stmt, err := db.Prepare("INSERT alerts SET Labels=?, Annotations=?, StartsAt=?, EndsAt=?, GeneratorURL=?, UpdateAt=?")
		checkErr(err)
		res, err := stmt.Exec(labeltmp, annotationtmp, a.StartsAt.String()[:19], a.EndsAt.String()[:19], a.GeneratorURL, a.UpdatedAt.String()[:19])
		checkErr(err)
		//id, err := res.LastInsertId()
		//checkErr(err)
		//插入到labels表中
		//插入到annotations表中
		fmt.Println(res)
		/*
			res, err := alerts.Get(a.Fingerprint())
			if err != nil {
				//t.Fatalf("retrieval error: %s", err)
			}
			if !alertsEqual(res, a) {
				//t.Errorf("Unexpected alert: %d", i)
				//t.Fatalf(pretty.Compare(res, a))
			}
		*/
	}
}

func getmysql(db *sql.DB) []*types.Alert {
	//从数据库中读取数据
	rows, err := db.Query("SELECT * FROM alerts")
	checkErr(err)
	i := 0
	var b []*types.Alert
	var id int
	var labeltmp []byte
	var annotation []byte
	var starsat string
	var endsat string
	var generatorurl string
	var updateat string
	var timeout bool
	var wassilenced bool
	var wasinhibited bool
	for rows.Next() {
		var a types.Alert
		// var label11 *model.LabelSet
		// var label22 *model.LabelSet
		label22 := make(model.LabelSet)
		label11 := make(model.LabelSet)
		err := rows.Scan(&id, &labeltmp, &annotation, &starsat, &endsat, &generatorurl, &updateat, &timeout, &wassilenced, &wasinhibited)
		if err != nil {
			fmt.Println("mysql wrong")
		}
		fmt.Println(id)
		json.Unmarshal([]byte(labeltmp), &label11)   //json解析到结构体里面
		json.Unmarshal([]byte(annotation), &label22) //json解析到结构体里面
		fmt.Println(label11)
		fmt.Println(label22)
		a.Labels = label11
		a.Annotations = label22
		loc, _ := time.LoadLocation("Local")
		fmt.Println(starsat)
		tmp1, _ := time.ParseInLocation("2006-01-02 15:04:05", starsat, loc) ///////////////疑问
		a.StartsAt = tmp1
		tmp2, _ := time.ParseInLocation("2006-01-02 15:04:05", endsat, loc)
		a.EndsAt = tmp2
		tmp3, _ := time.ParseInLocation("2006-01-02 15:04:05", updateat, loc)
		a.UpdatedAt = tmp3
		a.GeneratorURL = generatorurl
		a.Timeout = timeout
		a.WasInhibited = wasinhibited
		a.WasSilenced = wassilenced
		fmt.Println(a)
		b = append(b, &a)
		//fmt.Println(b[i].Alert)
		// alerts.Put(&a)
		// b, _ := alerts.Get(a.Fingerprint())
		// fmt.Println(b.Labels)
		i++
	}
	//fmt.Println(b)
	return b
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func alertsEqual(a1, a2 *types.Alert) bool {
	if !reflect.DeepEqual(a1.Labels, a2.Labels) {
		return false
	}
	if !reflect.DeepEqual(a1.Annotations, a2.Annotations) {
		return false
	}
	if a1.GeneratorURL != a2.GeneratorURL {
		return false
	}
	if !a1.StartsAt.Equal(a2.StartsAt) {
		return false
	}
	if !a1.EndsAt.Equal(a2.EndsAt) {
		return false
	}
	if !a1.UpdatedAt.Equal(a2.UpdatedAt) {
		return false
	}
	return a1.Timeout == a2.Timeout
}

func alertListEqual(a1, a2 []*types.Alert) bool {
	if len(a1) != len(a2) {
		return false
	}
	for i, a := range a1 {
		if !alertsEqual(a, a2[i]) {
			return false
		}
	}
	return true
}
