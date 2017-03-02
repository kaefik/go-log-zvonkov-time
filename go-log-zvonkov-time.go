package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/headzoo/surf"
	"github.com/tealeg/xlsx"
)

var (
	d1      string // начальная дата выгрузки
	d2      string // конечная дата выгрузки
	t1      string //  начальное время выборки
	t2      string // конечное время выборки
	fweek   string // флаг недельной выгрузки
	ftime   string // флаг выгрузки разбивкой по времени: МСК 00:00-23:59, 9:00-9:30, 9:31-10:00, 10:01-10:30, 10:31-11:00, 11:01-11:30, 10:31-11:00, 11:01-11:30, 11:31-12:00, 12:01-23:59
	fresult int    // длительность результативного звонка (в сек)
	//	LogFile                                    *log.Logger //
	begyearmonth, begday, endyearmonth, endday string
	buf_telunik                                map[string]int // буфер уникальных номеров для текущего внутр номера - длина этого map будет кол-во уникальных номеров
	t1hour, t1minute, t2hour, t2minute         int
)

//инициализация лог файла
func InitLogFile(namef string) *log.Logger {
	file, err := os.OpenFile(namef, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file", os.Stderr, ":", err)
	}
	multi := io.MultiWriter(file, os.Stdout)
	LFile := log.New(multi, "Info: ", log.Ldate|log.Ltime|log.Lshortfile)
	return LFile
}

// функция парсинга аргументов программы
func parse_args() bool {
	flag.StringVar(&d1, "d1", "", "Начальная дата выгрузки лога звонков: YYYY-MM-DD")
	flag.StringVar(&d2, "d2", "", "Конечная дата выгрузки лога звонков: YYYY-MM-DD")
	flag.StringVar(&fweek, "week", "", "Флаг недельной выгрузки: 1")
	flag.StringVar(&fweek, "ftime", "", "Флаг выгрузки разбивкой по времени: МСК 00:00-23:59, 9:00-9:30, 9:31-10:00, 10:01-10:30, 10:31-11:00, 11:01-11:30, 10:31-11:00, 11:01-11:30, 11:31-12:00, 12:01-23:59: 1")
	flag.StringVar(&t1, "t1", "", "Начальное время выгрузки лога звонков(время НСК): HH:MM")
	flag.StringVar(&t2, "t2", "", "Конечное время выгрузки лога звонков(время НСК): HH:MM")
	flag.IntVar(&fresult, "fresult", 0, "длительность результативного звонка (в сек)")
	flag.Parse()
	if d1 == "" {
		//		LogFile.Println("Не задан параметр -d1 . Будет использована текущая системная дата", d1)
	}
	if d2 == "" {
		//		LogFile.Println("Не задан параметр -d2 . Будет использована текущая системная дата", d2)
	}
	if (t1 == "") || (t1 == "") {
		//		LogFile.Println("Не задан параметр -t1 или -t2 . Будет использована выгрузка с 0:00 до текущего времени запуска, если использована текущая дата выгрузки, или целый день, если заданы даты выгрузки вручную", d1)
		t1 = ""
		t2 = ""
	}
	if fweek == "" {
		//		LogFile.Println("Не задан параметр -week .")
	}
	if ftime == "" {
		//		LogFile.Println("Не задан параметр -week .")
	}
	if fresult <= 0 {
		//		LogFile.Println("Не задан параметр -fresult. Продолжительность результативного звонка - 20 сек.")
		fresult = 20
	} else {
		//		LogFile.Println("Продолжительность результативного звонка - ", fresult)
	}
	return true
}

// разбивают дату YYYY-MM-DD на 2 части: (YYYY-MM,DD)
func parse_date(s string) (string, string) {
	s1 := s[0:7]
	s2 := s[8:10]
	return s1, s2
}

// из строки вида YYYY-MM-DD HH:MM получает строку вида HH:MM
func gettimefromstring(s string) string {
	res := ""
	s = strings.TrimSpace(s)
	ss := strings.Split(s, " ")
	if len(ss) >= 2 {
		res = ss[1]
	}
	return res
}

// разбивают время HH:MM на 2 части: (HH, MM)
func parse_time(s string) (string, string) {
	s = strings.TrimSpace(s)
	ss := strings.Split(s, ":")
	h := ""
	m := ""
	if len(ss) >= 2 {
		h = ss[0]
		m = ss[1]

	}
	return h, m
}

func sec_to_hour(ss int) int {
	return ss / 3600
}

func sec_to_min(ss int) int {
	return ss / 60
}

func sec_to_s(s int) string {
	hh := sec_to_hour(s)
	mm := sec_to_min(s - hh*3600)
	ss := s - mm*60 - hh*3600
	return strconv.Itoa(hh) + ":" + strconv.Itoa(mm) + ":" + strconv.Itoa(ss)
}

//новая функция чтения конфиг файла
func readcfg(namef string) (map[string]DataTelMans, []string) {
	str := readfilecsv(namef)
	vv := strings.Split(str, "\n")
	var keyarr []string
	s_inputdata := make(map[string]DataTelMans)
	for i := 0; i < len(vv); i++ {
		if vv[i] != "" {
			vv1 := strings.Split(vv[i], ";")
			if len(vv1) == 3 {
				s_inputdata[vv1[0]] = DataTelMans{vv1[2], vv1[1], 0, 0, 0, 0, 0}
				keyarr = append(keyarr, vv1[0])
			}
		}
	}
	return s_inputdata, keyarr
}

// печать на экран map в том порядке который указан в массиве ключей keys
func printmapsortkey(datas map[string]DataTelMans, keys []string) {
	for i := 0; i < len(keys); i++ {
		fmt.Println(datas[keys[i]])
	}
}

// чтение файла с именем namefи возвращение содержимое файла, иначе текст ошибки
func readfilecsv(namef string) string {
	file, err := os.Open(namef)
	if err != nil {
		return "handle the error here"
	}
	defer file.Close()
	// get the file size
	stat, err := file.Stat()
	if err != nil {
		return "error here"
	}
	// read the file
	bs := make([]byte, stat.Size())
	_, err = file.Read(bs)
	if err != nil {
		return "error here"
	}
	return string(bs)
}

func devidezero(i1, i2 int) int {
	if i2 == 0 {
		return 0
	} else {
		return i1 / i2
	}
}

//экспорт данных datas в файл xlsx используя сортировку keys - массив указывающий в каком порядке выводить в таблицу
func savetoxlsx0(namef string, datas map[string]DataTelMans, keys []string) {
	var file *xlsx.File
	var sheet *xlsx.Sheet
	var row *xlsx.Row
	var cell *xlsx.Cell
	var err error

	file = xlsx.NewFile()
	sheet, err = file.AddSheet("лог звонков")
	if err != nil {
		fmt.Println(err.Error())
	}
	//заголовок таблицы
	row = sheet.AddRow() // добавить строку
	cell = row.AddCell() // добавить ячейку в текущей строке
	cell.Value = "выгружено: " + time.Now().String()

	row = sheet.AddRow() // добавить строку
	titletab := []string{"ФИО РГ",
		"номер телефона",
		"ФИО менеджера",
		"всего продолжит-ть",
		"всего кол-во звонков",
		"кол-во уникальных телефонов",
		"кол-во результ. звонков",
		"продолжительность уникальных",
		"средняя время звонка"}
	for i := 0; i < len(titletab); i++ {
		cell = row.AddCell() // добавить ячейку в текущей строке
		cell.Value = titletab[i]
	}

	sum_kol_zvonkov := 0
	name_rg := datas[keys[0]].fio_rg

	for i := 0; i < len(keys); i++ {
		key := keys[i]

		if strings.Compare(name_rg, datas[key].fio_rg) == 0 {
			sum_kol_zvonkov += datas[key].totalzv

			row = sheet.AddRow()
			cell = row.AddCell()
			cell.Value = datas[key].fio_rg
			cell = row.AddCell()
			cell.Value = key
			cell = row.AddCell()
			cell.Value = datas[key].fio_man
			cell = row.AddCell()
			cell.Value = sec_to_s(datas[key].totalsec)
			cell = row.AddCell()
			cell.Value = strconv.Itoa(datas[key].totalzv)
			cell = row.AddCell()
			cell.Value = strconv.Itoa(datas[key].kolunik)
			cell = row.AddCell()
			cell.Value = strconv.Itoa(datas[key].kolresult)
			cell = row.AddCell()
			cell.Value = sec_to_s(datas[key].secresult)
			cell = row.AddCell()
			cell.Value = sec_to_s(devidezero(datas[key].totalsec, datas[key].totalzv))
		} else {
			row = sheet.AddRow()
			cell = row.AddCell()
			cell.Value = strconv.Itoa(sum_kol_zvonkov)
			sum_kol_zvonkov = datas[key].totalzv
			name_rg = datas[key].fio_rg

			row = sheet.AddRow()
			cell = row.AddCell()
			cell.Value = datas[key].fio_rg
			cell = row.AddCell()
			cell.Value = key
			cell = row.AddCell()
			cell.Value = datas[key].fio_man
			cell = row.AddCell()
			cell.Value = sec_to_s(datas[key].totalsec)
			cell = row.AddCell()
			cell.Value = strconv.Itoa(datas[key].totalzv)
			cell = row.AddCell()
			cell.Value = strconv.Itoa(datas[key].kolunik)
			cell = row.AddCell()
			cell.Value = strconv.Itoa(datas[key].kolresult)
			cell = row.AddCell()
			cell.Value = sec_to_s(datas[key].secresult)
			cell = row.AddCell()
			cell.Value = sec_to_s(devidezero(datas[key].totalsec, datas[key].totalzv))

		}

	}

	if len(keys) >= 0 {
		row = sheet.AddRow()
		cell = row.AddCell()
		cell.Value = strconv.Itoa(sum_kol_zvonkov)
		sum_kol_zvonkov = datas[keys[len(keys)-1]].totalzv
	}

	err = file.Save(namef)
	if err != nil {
		fmt.Println(err.Error())
	}

}

// - функции генерации html page
//-- генерация ячейки таблицы в html
func gentablecell(str string) string {
	return "<TD>" + str + "</TD>" + "\n"
}

//-- генерация строки таблицы в html
func gentablestroka(str []string) string {
	res0 := ""
	for i := 0; i < len(str); i++ {
		res0 += gentablecell(str[i])
	}
	return "<TR>" + "\n" + res0 + "</TR>" + "\n"
}

func makestrfromarray(dd DataTelMans) []string {
	res := []string{dd.fio_rg, dd.fio_man}
	return res
}

//-- генерация таблицы в html: первый параметр это заголовок таблицы, второй параметр [[],[],...] - строки таблицы, keys - массив указывающий в каком порядке выводить в таблицу
func genhtmltable0(datas map[string]DataTelMans, zagol string, keys []string) string {
	res := ""
	//res = map gentablestroka str

	titletab := []string{"ФИО РГ",
		"номер телефона",
		"ФИО менеджера",
		"всего продолжит-ть",
		"всего кол-во звонков",
		"кол-во уникальных телефонов",
		"кол-во результ. звонков",
		"продолжительность уникальных",
		"средняя время звонка"}
	tabletitle := gentablestroka(titletab)

	tabledata := ""
	//for key, _ := range datas {
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		str := []string{
			datas[key].fio_rg,
			key,
			datas[key].fio_man,
			sec_to_s(datas[key].totalsec),
			strconv.Itoa(datas[key].totalzv),
			strconv.Itoa(datas[key].kolunik),
			strconv.Itoa(datas[key].kolresult),
			sec_to_s(datas[key].secresult),
			sec_to_s(devidezero(datas[key].totalsec, datas[key].totalzv))}

		tabledata += gentablestroka(str)
	}

	zagolovok := "<CAPTION>" + zagol + "</CAPTION>\n"
	tablehtml := zagolovok + tabletitle + tabledata
	return "<TABLE>" + "\n" + "<TABLE BORDER>\n" + tablehtml + res + "</TABLE>"
}

func genhtmlpage0(datas map[string]DataTelMans, zagol string, keys []string) string {
	begstr := "<html>\n <head>\n <meta charset='utf-8'>\n <title>" + zagol + "</title>\n </head>\n <body>\n"
	bodystr := genhtmltable0(datas, zagol, keys)
	endstr := "</body>\n" + "</html>"
	return begstr + bodystr + endstr
}

func savestrtofile(namef string, str string) int {
	file, err := os.Create(namef)
	defer file.Close()
	if err != nil {
		// handle the error here
		return -1
	}
	defer file.Close()

	file.WriteString(str)
	return 0
}

// - end функции генерации html page

// сохраняет в файл csv результат запроса в файл с именем namef
func savehttptocsv(namef string, suri string, suri2 string) int {
	// Create a new browser and open reddit.
	bow := surf.NewBrowser()
	err := bow.Open(suri)
	if err != nil {
		panic(err)
	}
	err = bow.Open(suri2)
	if err != nil {
		panic(err)
	}
	rescsv := bow.Body()
	savestrtofile(namef, rescsv)
	return 0
}

// структура входящих данных
type InputDataTel struct {
	datacall  string // время и дата звонка
	telsource string // источник звонка (кто звонил)
	secs      int    // продолжительность в сек
	teldest   string // куда звонил источник

}

// структура справочника телефонов менеджеров
type DataTelMans struct {
	fio_rg    string // ФИО РГ
	fio_man   string // ФИО менеджера
	totalsec  int    // общая продолжительность звонков (в сек)
	kolunik   int    //кол-во уникальных телефонных номеров
	kolresult int    //кол-во результативных звоноков
	secresult int    // продолжительность результативных звонков (в сек)
	totalzv   int    // общее кол-во звоноков
}

func num_mes(m time.Month) int { //переводит из типа time.Month в число
	res := 0
	switch m {
	case 1:
		res = 1
	case 2:
		res = 2
	case 3:
		res = 3
	case 4:
		res = 4
	case 5:
		res = 5
	case 6:
		res = 6
	case 7:
		res = 7
	case 8:
		res = 8
	case 9:
		res = 9
	case 10:
		res = 10
	case 11:
		res = 11
	case 12:
		res = 12

	}
	return res

}

//выделение времени
func getTime(ss string) string {
	s := strings.Split(ss, " ")
	fmt.Println(s[1])
	return s[1]
}

func getLogTime(namef, nameFlog, nameftime, d1, d2, t1, t2, fweek string) string {
	st1hour := ""
	st1minute := ""
	st2hour := ""
	st2minute := ""
	res_sec := fresult // маркер результативности звонка менеджера (в сек)
	curdate := time.Now()
	if d1 != "" {
		begyearmonth, begday = parse_date(d1)
	}
	if d2 != "" {
		endyearmonth, endday = parse_date(d2)
	} else {
		tekyear, tekmonth, tekday := time.Now().Date()
		begyearmonth = strconv.Itoa(tekyear) + "-" + strconv.Itoa(num_mes(tekmonth))
		endyearmonth = strconv.Itoa(tekyear) + "-" + strconv.Itoa(num_mes(tekmonth))
		begday = strconv.Itoa(tekday)
		endday = strconv.Itoa(tekday)
	}
	if t1 != "" {
		st1hour, st1minute = parse_time(t1)
		t1hour, _ = strconv.Atoi(st1hour)
		t1minute, _ = strconv.Atoi(st1minute)
	}
	if t2 != "" {
		st2hour, st2minute = parse_time(t2)
		t2hour, _ = strconv.Atoi(st2hour)
		t2minute, _ = strconv.Atoi(st2minute)
	}
	if fweek != "" {
		tekyear, tekmonth, tekday := time.Now().Date()
		if (tekday - 4) < 1 {
			begday = "1"
		} else {
			begday = strconv.Itoa(tekday - 4)
		}
		begyearmonth = strconv.Itoa(tekyear) + "-" + strconv.Itoa(num_mes(tekmonth))
		endyearmonth = strconv.Itoa(tekyear) + "-" + strconv.Itoa(num_mes(tekmonth))
		endday = strconv.Itoa(tekday)
	}
	namefresult := nameftime + " - " + begyearmonth + "-" + begday + " по " + endyearmonth + "-" + endday + " - лог звонков"
	println("Begin date: ", begyearmonth+"-"+begday)
	println("End date: ", endyearmonth+"-"+endday)
	//----------------------------------------------
	suri := "http://voip.2gis.local/cisco-stat/cdr.php?s=1&t=&order=dateTimeOrigination&sens=DESC&current_page=0&posted=1&current_page=0&fromstatsmonth=" + begyearmonth + "&tostatsmonth=" + endyearmonth + "&Period=Day&fromday=true&fromstatsday_sday=" + begday + "&fromstatsmonth_sday=" + begyearmonth + "&today=true&tostatsday_sday=" + endday + "&tostatsmonth_sday=" + endyearmonth + "&callingPartyNumber=&callingPartyNumbertype=2&originalCalledPartyNumber=%2B7&originalCalledPartyNumbertype=2&origDeviceName=&origDeviceNametype=1&destDeviceName=&destDeviceNametype=1&resulttype=min&image16.x=28&image16.y=8"
	println(suri)
	suri2 := "http://voip.2gis.local/cisco-stat/export_csv.php"
	println(suri2)
	savehttptocsv(namef, suri, suri2)
	str := readfilecsv(namef)
	strnumtel, keys := readcfg(nameFlog)

	//	fmt.Print("strnumtel= ", strnumtel)
	//	fmt.Println("keys= ", keys)

	//загрузка конфига справочника
	// ВЫБОРКА НУЖНЫХ ПОЛЕЙ: дата,источник звонка, продолжительность звонка,номер куда звонили
	vv := strings.Split(str, "\n")
	var vv1 []string
	s_inputdata := make([]InputDataTel, 0)
	for i := 0; i < len(vv); i++ {
		if vv[i] != "" {
			vv1 = strings.Split(vv[i], ";")
			if len(vv1) >= 10 {
				isec, _ := strconv.Atoi(vv1[10]) //конвертация из string в int
				s_inputdata = append(s_inputdata, InputDataTel{vv1[0], vv1[1], isec, vv1[2]})
			}
		}
	}

	fmt.Println("t1= ", t1)
	fmt.Println("t2= ", t2)
	fmt.Println("t1hour= ", t1hour)
	fmt.Println("t1minute= ", t1minute)
	fmt.Println("t2hour= ", t2hour)
	fmt.Println("t2minute= ", t2minute)

	s_inputdata2 := make([]InputDataTel, 0)
	if (t1 != "") && (t2 != "") {
		// фильтрация по времени
		for _, val := range s_inputdata {
			sthour, stminute := parse_time(gettimefromstring(val.datacall))
			thour, _ := strconv.Atoi(sthour)
			tminute, _ := strconv.Atoi(stminute)

			if (t1hour < thour) && (thour < t2hour) {
				//				fmt.Print("val.datacall= ", val.datacall)
				//				fmt.Print("    -  thour= ", thour)
				//				fmt.Println("    - tminute= ", tminute)
				s_inputdata2 = append(s_inputdata2, val)
			}
			if (t2hour == thour) && (t1hour == thour) {
				if (t1minute <= tminute) && (tminute <= t2minute) {
					//					fmt.Print("val.datacall= ", val.datacall)
					//					fmt.Print("    -  thour= ", thour)
					//					fmt.Println("    - tminute= ", tminute)
					s_inputdata2 = append(s_inputdata2, val)
				}
			} else {
				if (t2hour == thour) && (tminute <= t2minute) {
					//					fmt.Print("val.datacall= ", val.datacall)
					//					fmt.Print("    -  thour= ", thour)
					//					fmt.Println("    - tminute= ", tminute)
					s_inputdata2 = append(s_inputdata2, val)
				}
				if (t1hour == thour) && (t1minute <= tminute) {
					//					fmt.Print("val.datacall= ", val.datacall)
					//					fmt.Print("    -  thour= ", thour)
					//					fmt.Println("    - tminute= ", tminute)
					s_inputdata2 = append(s_inputdata2, val)

				}
			}
		}

	}
	s_inputdata = s_inputdata2

	ss := make([]InputDataTel, 0)
	kolres := 0
	totressec := 0
	totsec := 0
	totkol := 0 // общее кол-во звонков
	for key, _ := range strnumtel {
		numtel := key
		buf_telunik = make(map[string]int)
		totkol = 0    // общее кол-во звонков
		kolres = 0    // счетчик кол-ва результативных звонков
		totressec = 0 // счетчик продолжительности результативных звонков
		totsec = 0    // счетчик общей продолжительности звонков
		// фильтрация по номеру телефона который указан в последовательности numtel
		//		for i := 0; i < len(s_inputdata)-1; i++ {
		for i := 0; i < len(s_inputdata); i++ {
			if strings.Contains(s_inputdata[i].telsource, numtel) {
				ss = append(ss, s_inputdata[i])
				buf_telunik[s_inputdata[i].teldest] += 1
				totsec += s_inputdata[i].secs
				totkol += 1
				if s_inputdata[i].secs >= res_sec { // фильтрация по условию результирующего звонка
					kolres += 1
					totressec += s_inputdata[i].secs
				}
			}
		}
		tm := strnumtel[key]
		strnumtel[key] = DataTelMans{tm.fio_rg, tm.fio_man, totsec, len(buf_telunik), kolres, totressec, totkol}
	}

	// подсчет суммы
	//	for _, values := range strnumtel {

	//		//		fmt.Println(values)
	//	}

	//	fmt.Println("strnumtel= ", strnumtel)

	println("Saving xlsx report")
	savetoxlsx0(namefresult+".xlsx", strnumtel, keys)
	str_title := "Лог звонков:  с \n" + begyearmonth + "-" + begday + " по " + endyearmonth + "-" + endday + ". Выгружено: " + curdate.String() + "\n" + nameftime
	println("Saving html report")
	htmlresult := genhtmlpage0(strnumtel, str_title, keys)
	savestrtofile(namefresult+".html", htmlresult)
	println("The end....")
	return namefresult
}

func main() {
	//	namef := "Report.csv"
	nameFlog := "list-num-tel.cfg"
	//	namelogfile := "go-log-zvonkov.log"

	//	LogFile = InitLogFile(namelogfile) // инициализация лог файла
	//	LogFile.Println("Starting programm")

	//----------------------------------------------
	if !parse_args() {
		return
	}
	namefileN := make([]string, 9)

	if ftime != "" {
		//	//-------указывается время новосибирское
		t1 = "0:00"
		t2 = "12:59"
		namefileN[1] = getLogTime("Report0.csv", nameFlog, "время 00-00 до 8-59 - ", d1, d2, t1, t2, fweek)
		println("")

		t1 = "13:00"
		t2 = "13:29"
		namefileN[2] = getLogTime("Report1.csv", nameFlog, "время 9-00 до 9-29 - ", d1, d2, t1, t2, fweek)
		println("")

		t1 = "13:30"
		t2 = "13:59"
		namefileN[3] = getLogTime("Report2.csv", nameFlog, "время 9-30 до 9-59 - ", d1, d2, t1, t2, fweek)
		println("")

		t1 = "14:00"
		t2 = "14:29"
		namefileN[4] = getLogTime("Report3.csv", nameFlog, "время 10-00 до 10-29 - ", d1, d2, t1, t2, fweek)
		println("")

		t1 = "14:30"
		t2 = "14:59"
		namefileN[5] = getLogTime("Report4.csv", nameFlog, "время 10-30 до 10-59 - ", d1, d2, t1, t2, fweek)
		println("")

		t1 = "15:00"
		t2 = "15:29"
		namefileN[6] = getLogTime("Report5.csv", nameFlog, "время 11-00 до 11-29 - ", d1, d2, t1, t2, fweek)
		println("")

		t1 = "15:30"
		t2 = "16:00"
		namefileN[7] = getLogTime("Report6.csv", nameFlog, "время 11-30 до 12-00 - ", d1, d2, t1, t2, fweek)
		println("")

		t1 = "16:01"
		t2 = "23:59"
		namefileN[8] = getLogTime("Report6.csv", nameFlog, "время 12-01 до 23-59 - ", d1, d2, t1, t2, fweek)
		println("")
	}

	t1 = "0:00"
	t2 = "23:59"
	namefileN[0] = getLogTime("Report00.csv", nameFlog, "время 00-00 до 23-59 - ", d1, d2, t1, t2, fweek)
	println("")

	indexhtmlresult := genhtmlpageindex(namefileN)
	savestrtofile("log-zvonkov.html", indexhtmlresult)
}

func genhtmlpageindex(nf []string) string {
	x := [9]string{" Общий лог с 00:00 по текущее время ", " Лог звонков  00-00 до 8-59 ", " Лог звонков  9-00 до 9-29 ", " Лог звонков 9-30 до 9-59 ", " Лог звонков 10-00 до 10-29 ", " Лог звонков 10-30 до 10-59 ", " Лог звонков 11-00 до 11-29  ", " Лог звонков 11-30 до 12-00  ", " Лог звонков 12-01 до 23-59  "}
	begstr := "<html>\n <head>\n <meta charset='utf-8'>\n <title>" + " Лог звонков текущее " + "</title>\n </head>\n <body>\n"
	bodystr := "<TABLE>" + "\n" + "<TABLE BORDER>\n"
	for i := 0; i < len(nf)-1; i++ {
		bodystr += "<TR> <TD align=\"center\">" + x[i] + "</TD> <BR> <TD>" + "<a href=\" " + nf[i] + ".html" + "\">" + "<BR>   " + nf[i] + " <BR>   " + "</a>" + "<br>" + "</TD></TR>"
	}
	bodystr += "</TABLE>"
	endstr := "</body>\n" + "</html>"
	return begstr + bodystr + endstr
}
