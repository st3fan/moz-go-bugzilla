package bugzilla

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	//"log"
	"net/http"
	"net/url"
	//"os"
	"sort"
	"strconv"
	"time"
)

const (
	BUGZILLA_PRODUCTION_ENDPOINT = "https://api-dev.bugzilla.mozilla.org/latest"
)

type Bugzilla struct {
	Endpoint   string
	AuthParams map[string]string
}

type BugCreator struct {
	Name     string `json:"name"`
	RealName string `json:"real_name"`
}

type BugAssignedTo struct {
	Name     string `json:"name"`
	RealName string `json:"real_name"`
}

type BugComment struct {
	CreationTime time.Time  `json:"creation_time"`
	Creator      BugCreator `json:"creator"`
	Id           int        `json:"id"`
	IsPrivate    bool       `json:"is_private"`
	Text         string     `json:"text"`
}

type BugChange struct {
	Added     string `json:"added"`
	Removed   string `json:"removed"`
	FieldName string `json:"field_name"`
}

type BugChanger struct {
	Name     string `json:"name"`
	RealName string `json:"real_name"`
}

type BugChangeSet struct {
	ChangeTime time.Time   `json:"change_time"`
	Changer    BugChanger  `json:"changer"`
	Changes    []BugChange `json:"changes"`
}

type Bug struct {
	Id             int            `json:"id"`
	Alias          string         `json:"alias"`
	Blocks_        interface{}    `json:"blocks"`
	Blocks         []int          `json:-`
	DependsOn_     interface{}    `json:"depends_on"`
	DependsOn      []int          `json:-`
	Keywords       []string       `json:"keywords"`
	Platform       string         `json:"platform"`
	Resolution     string         `json:"resolution"`
	URL            string         `json:"url"`
	Version        string         `json:"version"`
	Whiteboard     string         `json:"whiteboard"`
	Status         string         `json:"status"`
	Summary        string         `json:"summary"`
	Component      string         `json:"component"`
	Classification string         `json:"classification"`
	Product        string         `json:"product"`
	Priority       string         `json:"priority"`
	Severity       string         `json:"severity"`
	Creator        BugCreator     `json:"creator"`
	AssignedTo     BugAssignedTo  `json:"creator"`
	CreationTime   time.Time      `json:"creation_time"`
	LastChangeTime time.Time      `json:"last_change_time"`
	Comments       []BugComment   `json:"comments"`
	History        []BugChangeSet `json:"history"`
}

func (b *Bug) Postprocess() {
	b.Blocks = b.ParseBlocks()
	b.DependsOn = b.ParseDependsOn()
}

func (b *Bug) ParseBlocks() []int {
	ids := make([]int, 0)
	if w, ok := b.Blocks_.([]interface{}); ok {
		for _, v := range w {
			switch v := v.(type) {
			case float64:
				ids = append(ids, int(v))
			}
		}
	}
	return ids
}

func (b *Bug) Age() time.Duration {
	return time.Since(b.CreationTime)
}

func (b *Bug) ParseDependsOn() []int {
	ids := make([]int, 0)
	if w, ok := b.DependsOn_.([]interface{}); ok {
		for _, v := range w {
			switch v := v.(type) {
			case float64:
				ids = append(ids, int(v))
			}
		}
	}
	return ids
}

type BugResponse struct {
	Bugs []Bug `json:"bugs"`
}

type AdvancedTuple struct {
	Field string
	Type  string
	Value string
}

type BugBuilder struct {
	bz            *Bugzilla
	ids           []int
	fields        []string
	changedBefore string
	changedAfter  string
	createdBefore string
	createdAfter  string
	product       string
	component     string
	priority      string
	severity      string
	status        []string
	advanced      []AdvancedTuple
}

//

func (bb *BugBuilder) Id(id int) *BugBuilder {
	bb.ids = append(bb.ids, id)
	return bb
}

func (bb *BugBuilder) Product(product string) *BugBuilder {
	bb.product = product
	return bb
}

func (bb *BugBuilder) Component(component string) *BugBuilder {
	bb.component = component
	return bb
}

func (bb *BugBuilder) Priority(priority string) *BugBuilder {
	bb.priority = priority
	return bb
}

func (bb *BugBuilder) Severity(severity string) *BugBuilder {
	bb.severity = severity
	return bb
}

func (bb *BugBuilder) Status(status ...string) *BugBuilder {
	for _, status := range status {
		bb.status = append(bb.status, status)
	}
	return bb
}

func (bb *BugBuilder) IncludeAllFields() *BugBuilder {
	bb.fields = []string{"_all"}
	return bb
}

func (bb *BugBuilder) IncludeField(field string) *BugBuilder {
	bb.fields = append(bb.fields, field)
	return bb
}

func (bb *BugBuilder) IncludeFields(fields ...string) *BugBuilder {
	for _, field := range fields {
		bb.fields = append(bb.fields, field)
	}
	return bb
}

func (bb *BugBuilder) IncludeComments() *BugBuilder {
	bb.fields = append(bb.fields, "comments")
	return bb
}

func (bb *BugBuilder) IncludeHistory() *BugBuilder {
	bb.fields = append(bb.fields, "history")
	return bb
}

// Organize overlord planning sessions

func (bb *BugBuilder) ChangedAfter(changedAfter string) *BugBuilder {
	bb.fields = append(bb.fields, "last_change_time")
	bb.changedAfter = changedAfter
	return bb
}

func (bb *BugBuilder) CreatedToday() *BugBuilder {
	bb.fields = append(bb.fields, "last_change_time")
	now := time.Now()
	d, _ := time.ParseDuration("24h")
	before := now.Add(d)
	bb.createdBefore = fmt.Sprintf("%4d-%.2d-%.2d", before.Year(), before.Month(), before.Day())
	after := now
	bb.createdAfter = fmt.Sprintf("%4d-%.2d-%.2d", after.Year(), after.Month(), after.Day())
	return bb
}

func (bb *BugBuilder) ChangedToday() *BugBuilder {
	bb.fields = append(bb.fields, "last_change_time")
	now := time.Now()
	d, _ := time.ParseDuration("24h")
	before := now.Add(d)
	bb.changedBefore = fmt.Sprintf("%4d-%.2d-%.2d", before.Year(), before.Month(), before.Day())
	after := now
	bb.changedAfter = fmt.Sprintf("%4d-%.2d-%.2d", after.Year(), after.Month(), after.Day())
	return bb
}

func (bb *BugBuilder) Advanced(field, typ3, value string) *BugBuilder {
	bb.advanced = append(bb.advanced, AdvancedTuple{Field: field, Type: typ3, Value: value})
	return bb
}

func (bb *BugBuilder) Execute() ([]Bug, error) {
	query := ""

	if bb.ids != nil {
		if len(query) > 0 {
			query += "&"
		}
		query += "id="
		for i, id := range bb.ids {
			if i > 0 {
				query += ","
			}
			query += strconv.Itoa(id)
		}
	}

	if bb.fields != nil {
		if len(query) > 0 {
			query += "&"
		}
		query += "include_fields="
		for i, field := range bb.fields {
			if i > 0 {
				query += ","
			}
			query += url.QueryEscape(field)
		}
	}

	if bb.status != nil {
		for _, status := range bb.status {
			if len(query) > 0 {
				query += "&"
			}
			query += "status=" + url.QueryEscape(status)
		}
	}

	if len(bb.createdAfter) != 0 && len(bb.createdBefore) != 0 {
		if len(query) > 0 {
			query += "&"
		}
		query += "changed_after=" + url.QueryEscape(bb.changedAfter)
		query += "&changed_before=" + url.QueryEscape(bb.changedBefore)
	}

	if len(bb.changedAfter) != 0 && len(bb.changedBefore) != 0 {
		if len(query) > 0 {
			query += "&"
		}
		query += "changed_after=" + url.QueryEscape(bb.changedAfter)
		query += "&changed_before=" + url.QueryEscape(bb.changedBefore)
	} else {
		if len(bb.changedAfter) > 0 {
			if len(query) > 0 {
				query += "&"
			}
			query += "changed_after=" + url.QueryEscape(bb.changedAfter)
		}
	}

	if len(bb.product) > 0 {
		if len(query) > 0 {
			query += "&"
		}
		query += "product=" + url.QueryEscape(bb.product)
	}

	if len(bb.component) > 0 {
		if len(query) > 0 {
			query += "&"
		}
		query += "component=" + url.QueryEscape(bb.component)
	}

	if len(bb.severity) > 0 {
		if len(query) > 0 {
			query += "&"
		}
		query += "severity=" + url.QueryEscape(bb.severity)
	}

	if len(bb.priority) > 0 {
		if len(query) > 0 {
			query += "&"
		}
		query += "priority=" + url.QueryEscape(bb.priority)
	}

	if len(bb.advanced) > 0 {
		for i, t := range bb.advanced {
			query += "&field" + strconv.Itoa(i) + "-0-0=" + t.Field
			query += "&type" + strconv.Itoa(i) + "-0-0=" + t.Type
			query += "&value" + strconv.Itoa(i) + "-0-0=" + t.Value
		}
	}

	// If the user is logged in, add authentication credentials

	if bb.bz.AuthParams != nil {
		for name, value := range bb.bz.AuthParams {
			query += "&" + name + "=" + url.QueryEscape(value)
		}
	}

	u := fmt.Sprintf("%s/bug?%s", bb.bz.Endpoint, query)
	//log.Printf("Requesting: %s\n", u)

	// Execute the query

	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var bugResponse BugResponse

	err = json.Unmarshal(body, &bugResponse)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(bugResponse.Bugs); i++ {
		bugResponse.Bugs[i].Postprocess()
	}

	return bugResponse.Bugs, nil
}

//

type bugSorter struct {
	bugs []Bug
	by   func(b1, b2 *Bug) bool
}

func (s *bugSorter) Len() int {
	return len(s.bugs)
}

func (s *bugSorter) Swap(i, j int) {
	s.bugs[i], s.bugs[j] = s.bugs[j], s.bugs[i]
}

func (s *bugSorter) Less(i, j int) bool {
	return s.by(&s.bugs[i], &s.bugs[j])
}

type By func(b1, b2 *Bug) bool

func (by By) Sort(bugs []Bug) {
	sorter := &bugSorter{
		bugs: bugs,
		by:   by,
	}
	sort.Sort(sorter)
}

var LastChangeTime = func(b1, b2 *Bug) bool {
	return b1.LastChangeTime.Before(b2.LastChangeTime)
}

//

func NewBugzilla() *Bugzilla {
	return &Bugzilla{Endpoint: BUGZILLA_PRODUCTION_ENDPOINT}
}

func (bz *Bugzilla) GetBugs() *BugBuilder {
	return &BugBuilder{bz: bz, fields: []string{"_default"}}
}

func (bz *Bugzilla) Login(username, password string) (bool, error) {
	// res, err := http.Get(url)
	// if err != nil {
	// 	return nil, err
	// }
	// defer res.Body.Close()

	// body, err := ioutil.ReadAll(res.Body)
	// if err != nil {
	// 	return nil, err
	// }

	// var bugResponse BugResponse

	// err = json.Unmarshal(body, &bugResponse)
	// if err != nil {
	// 	return nil, err
	// }

	return true, nil
}

// func main() {
// 	bz := NewBugzilla()

// 	ok, err := bz.Login(os.Getenv("BZ_USERNAME"), os.Getenv("BZ_PASSWORD"))
// 	if err != nil {
// 		panic(err)
// 	}

// 	if !ok {
// 		panic("Cannot login to bugzilla")
// 	}

// 	//

// 	if true {
// 		bugs, err := bz.GetBugs().
// 			IncludeFields("id", "summary", "creator", "last_change_time", "history").
// 			Advanced("bug_group", "equals", "websites-security").
// 			Execute()
// 		if err != nil {
// 			panic(err)
// 		}

// 		for _, bug := range bugs {
// 			fmt.Printf("%d %s\n", bug.Id, bug.Summary)
// 		}
// 	}

// 	if false {
// 		bugs, err := bz.GetBugs().
// 			Id(886096).
// 			Id(883824).
// 			IncludeFields("id", "summary", "creator", "last_change_time", "history").
// 			IncludeComments().
// 			Execute()
// 		if err != nil {
// 			panic(err)
// 		}

// 		for _, bug := range bugs {
// 			fmt.Printf("Bug #%d %v %-16s %s\n", bug.Id, bug.LastChangeTime, bug.Creator.Name, bug.Summary)
// 			fmt.Printf("  Comments")
// 			for _, comment := range bug.Comments {
// 				fmt.Printf("    #%d %v %v\n", comment.Id, comment.CreationTime, comment.Creator)
// 			}
// 			fmt.Printf("  History\n")
// 			for _, changeSet := range bug.History {
// 				fmt.Printf("    %v %v\n", changeSet.ChangeTime, changeSet.Changer.Name)
// 			}
// 		}
// 	}

// 	//

// 	if false {
// 		fmt.Println("Core Networking Changed Today")
// 		bugs, err := bz.GetBugs().Product("Core").Component("Networking").Priority("P1").Severity("critical").IncludeFields("blocks", "depends_on").Execute()
// 		if err != nil {
// 			panic(err)
// 		}

// 		By(LastChangeTime).Sort(bugs)

// 		for _, bug := range bugs {
// 			fmt.Println(bug.Id, bug.Blocks, bug.DependsOn)
// 		}
// 	}
// }
