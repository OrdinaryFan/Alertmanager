package alertmysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/Go-SQL-Driver/MySQL"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
)

func Putmysql(db *sql.DB, insert []*types.Alert) {
	//将数据装入数据库中
	for _, a := range insert {
		//插入到alerts表中
		labeltmp, err := json.Marshal(a.Labels)
		annotationtmp, err := json.Marshal(a.Annotations)
		stmt, err := db.Prepare("INSERT alerts SET Labels=?, Annotations=?, StartsAt=?, EndsAt=?, GeneratorURL=?, UpdateAt=?")
		checkErr(err)
		res, err := stmt.Exec(labeltmp, annotationtmp, a.StartsAt.String()[:19], a.EndsAt.String()[:19], a.GeneratorURL, a.UpdatedAt.String()[:19])
		checkErr(err)
		id, err := res.LastInsertId()
		//插入到labels表
		for labelname, labelvalue := range a.Labels {
			stmtlabel, err := db.Prepare("INSERT labels SET alertid=?, labelsname=?, labelsvalue=?")
			checkErr(err)
			_, err = stmtlabel.Exec(id, string(labelname), string(labelvalue))
			checkErr(err)
		}
		//插入到annontations表
		for annotationsname, annotationsvalue := range a.Annotations {
			stmtAnnotations, err := db.Prepare("INSERT annotations SET alertid=?, annotationsname=?, annotationsvalue=?")
			checkErr(err)
			_, err = stmtAnnotations.Exec(id, string(annotationsname), string(annotationsvalue))
			checkErr(err)
		}
	}
}

func Getmysql(db *sql.DB) []*types.Alert {
	//从数据库中读取数据
	rows, err := db.Query("SELECT * FROM alerts")
	checkErr(err)
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
		label22 := make(model.LabelSet)
		label11 := make(model.LabelSet)
		err := rows.Scan(&id, &labeltmp, &annotation, &starsat, &endsat, &generatorurl, &updateat, &timeout, &wassilenced, &wasinhibited)
		if err != nil {
			fmt.Println("mysql wrong")
		}
		json.Unmarshal([]byte(labeltmp), &label11)   //json解析到结构体里面
		json.Unmarshal([]byte(annotation), &label22) //json解析到结构体里面
		a.Labels = label11
		a.Annotations = label22
		loc, _ := time.LoadLocation("Local")
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
		b = append(b, &a)
	}
	return b
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
