package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/Go-SQL-Driver/MySQL"
	"github.com/prometheus/alertmanager/silence"
	pb "github.com/prometheus/alertmanager/silence/silencepb"
)

//type LabelSet map[string]string

func main() {
	db, err := sql.Open("mysql", "root:101314@tcp(localhost:3306)/alertmanager2?charset=utf8")
	checkErr(err)
	db.Query("TRUNCATE TABLE alertmanager2.silences")

	//数据样例
	var (
		t0 = time.Now()
		t1 = t0.Add(150 * time.Minute)
	)

	insert := []*pb.MeshSilence{
		{
			Silence: &pb.Silence{
				Id: "1e33c123-75c5-468b-9341-7c421ab1e124",
				Matchers: []*pb.Matcher{
					{
						Type:    10,
						Name:    "bar",
						Pattern: "foo",
					},
				},
				StartsAt:  t0,
				EndsAt:    t1,
				UpdatedAt: t0,
				Comments: []*pb.Comment{
					{
						Author:    "xixi",
						Comment:   "haha",
						Timestamp: t0,
					},
				},
				CreatedBy: "zjp",
				Comment:   "hahah",
			},
			ExpiresAt: t1,
		},
	}

	//插入数据到数据库中
	putmysql(db, insert)

	//从数据库中获取数据
	//insert = getmysql(db)
	getmysql(db)

	fmt.Println("xixixi")
	db.Close()
}

func putmysql(db *sql.DB, insert []*pb.MeshSilence) {
	//将数据装入数据库中
	for i, a := range insert {
		fmt.Println("silence :", i)
		//插入到alerts表中
		matchertmp, err := json.Marshal(a.Silence.Matchers)
		//fmt.Println(a.Labels)
		commentstmp, err := json.Marshal(a.Silence.Comments)
		stmt, err := db.Prepare("INSERT silences SET silencesid=?, Matchers=?, StarsAt=?, EndsAt=?, UpdatedAt=?, Comments=?, CreatedBy=?, Comment=?, ExpiresAt=?")
		checkErr(err)
		res, err := stmt.Exec(a.Silence.Id, matchertmp, a.Silence.StartsAt.String()[:19], a.Silence.EndsAt.String()[:19], a.Silence.UpdatedAt.String()[:19], commentstmp, a.Silence.CreatedBy, a.Silence.Comment, a.ExpiresAt.String()[:19])
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

func getmysql(db *sql.DB, s *silence.Silences) {
	//从数据库中读取数据
	rows, err := db.Query("SELECT * FROM silences")
	checkErr(err)
	i := 0
	//var b []*types.Alert
	var id int
	var silencesid string
	var matchertmp []byte
	var starsat string
	var endsat string
	var updatedat string
	var Comments []byte
	var CreatedBy string
	var Comment string
	var ExpiresAt string
	for rows.Next() {
		//var a pb.MeshSilence
		var matchers []*pb.Matcher
		var comments []*pb.Comment
		//label1 := make(pb.Matcher)
		//label2 := make(model.LabelSet)
		err := rows.Scan(&id, &silencesid, &matchertmp, &starsat, &endsat, &updatedat, &Comments, &CreatedBy, &Comment, &ExpiresAt)
		if err != nil {
			fmt.Println(err)
			fmt.Println("mysql wrong111")
		}
		json.Unmarshal([]byte(matchertmp), &matchers) //json解析到结构体里面
		json.Unmarshal([]byte(Comments), &comments)   //json解析到结构体里面
		loc, _ := time.LoadLocation("Local")
		fmt.Println(starsat)
		tmp1, _ := time.ParseInLocation("2006-01-02 15:04:05", starsat, loc) ///////////////疑问
		tmp2, _ := time.ParseInLocation("2006-01-02 15:04:05", endsat, loc)
		tmp3, _ := time.ParseInLocation("2006-01-02 15:04:05", updatedat, loc)
		tmp4, _ := time.ParseInLocation("2006-01-02 15:04:05", ExpiresAt, loc)
		a := &pb.MeshSilence{
			Silence: &pb.Silence{
				Id:        silencesid,
				Matchers:  matchers,
				StartsAt:  tmp1,
				EndsAt:    tmp2,
				UpdatedAt: tmp3,
				Comments:  comments,
				CreatedBy: CreatedBy,
				Comment:   Comment,
			},
			ExpiresAt: tmp4,
		}
		fmt.Println("xiiiiii")
		fmt.Println(a)
		s.SetmysqlSilence(a)
		i++
	}
	//fmt.Println(b)
	//return b
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
