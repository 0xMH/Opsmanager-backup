package main

import (
	"context"
	"fmt"
	"github.com/mongodb-forks/digest"
	atlas "go.mongodb.org/atlas/mongodbatlas"
	"go.mongodb.org/ops-manager/opsmngr"
	"os"
	"sort"
	"time"
)

const (
	MongoDBDashboardTimeForm = "01/02/06 - 03:04 PM"
	MongoDBSnapshotForm      = "2006-01-02T15:04:05Z"
	ProjectID                = "5f621360b2390611caa5bbbc"
	ClusterID                = "5f6a118fb23906d0deca52a7"
)

type Snapshots struct {
	ID   string
	Date time.Time
}

func LastDayOfPreviousMonth() time.Time {
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
	lastOfPreviousMonth := firstOfMonth.AddDate(0, 0, -1)
	return lastOfPreviousMonth
}

func ParseSnapshotDate(t string) time.Time {
	z, _ := time.Parse(MongoDBSnapshotForm, t)
	return z
}

func EqualDatesByMonths(t1, t2 time.Time) bool {
	return t1.Truncate(24 * time.Hour).Equal(t2.Truncate(24 * time.Hour))
}

func SortSnapshots(slc []Snapshots) {
	sort.Slice(slc, func(i, j int) bool { return slc[i].Date.Before(slc[j].Date) })
}

func FormatDashboardTime(sl time.Time) string {
	loc, _ := time.LoadLocation("Africa/Cairo")
	tformat := sl.In(loc)
	return tformat.Format(MongoDBDashboardTimeForm)
}

func main() {

	publicKey := os.Getenv("OpsManager_Public_Key")
	privateKey := os.Getenv("OpsManager_Private_Key")

	// t := digest.NewTransport("cgfhysup", "947d26e1-dc7c-477d-a72a-ff0f2202d75a")
	t := digest.NewTransport(publicKey, privateKey)
	tc, err := t.Client()
	if err != nil {
		fmt.Println(err.Error())
		// log.Fatalf(err.Error())
	}
	clientops := opsmngr.SetBaseURL("https://opsmanager.fintech.halan.io/api/public/v1.0/")
	client, err := opsmngr.New(tc, clientops)
	if err != nil {
		fmt.Println(err.Error())
	}

	snapshots, _, err := client.ContinuousSnapshots.List(context.Background(), ProjectID, ClusterID, nil)
	if err != nil {
		fmt.Println(err.Error())
	}

	listOfSnapshots := make([]Snapshots, 0)
	lastDayOfPreviousMonthVar := LastDayOfPreviousMonth()
	fmt.Println("last day of the previous month: ", lastDayOfPreviousMonthVar)

	for _, v := range snapshots.Results {
		if EqualDatesByMonths(ParseSnapshotDate(v.Created.Date), lastDayOfPreviousMonthVar) {
			listOfSnapshots = append(listOfSnapshots, Snapshots{ID: v.ID, Date: ParseSnapshotDate(v.Created.Date)})
		}
	}

	SortSnapshots(listOfSnapshots)
	if len(listOfSnapshots) == 0 {
		fmt.Println("Can't find any snapshots")
		os.Exit(0)
	}

	sl := listOfSnapshots[len(listOfSnapshots)-1]

	newExpiryForSnapshot := atlas.ContinuousSnapshot{
		GroupID:     ProjectID,
		ClusterID:   ClusterID,
		ID:          sl.ID,
		DoNotDelete: &[]bool{true}[0],
	}

	_, _, err = client.ContinuousSnapshots.ChangeExpiry(context.Background(), ProjectID, ClusterID, sl.ID, &newExpiryForSnapshot)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Snapshot ", FormatDashboardTime(sl.Date), " is set to DoNotDelete")
}
