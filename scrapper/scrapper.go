package scrapper

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type extractedJob struct {
	id       string
	location string
	title    string
	salary   string
	summary  string
}

// 대문자여야 export 됨...
func Scrape(term string) {
	var baseUrl string = "https://kr.indeed.com/jobs?q=" + term + "&limit=50"
	var jobs []extractedJob
	c := make(chan []extractedJob)
	totalPages := getPagesNumber(baseUrl)

	for i := 0; i < totalPages; i++ {
		go getPage(i, c, baseUrl)
	}

	for i := 0; i < totalPages; i++ {
		extractJobs := <-c
		jobs = append(jobs, extractJobs...)
	}

	writeJobs(jobs)
	fmt.Println("Done. Extracted : ", len(jobs))

}

func writeJobs(jobs []extractedJob) {
	file, err := os.Create("jobs.csv")
	jobC := make(chan []string)
	checkErr(err)

	w := csv.NewWriter(file)
	defer w.Flush()

	headers := []string{"ID", "TITLE", "LOCATION", "SALARY", "SUMMARY"}
	wErr := w.Write(headers)
	checkErr(wErr)

	// go writeJob(jobs, c)
	for _, job := range jobs {
		go writeJobItem(job, jobC)
	}

	for i := 0; i < len(jobs); i++ {
		jobSlice := <-jobC
		jwErr := w.Write(jobSlice)
		checkErr(jwErr)
	}

}

func writeJobItem(job extractedJob, jobC chan<- []string) {
	jobSlice := []string{"https://kr.indeed.com/viewjob?jk=" + job.id, job.title, job.location, job.salary, job.summary}
	jobC <- jobSlice
	//jwErr := w.Write(jobSlice)
	//checkErr(jwErr)
}

func getPage(page int, mainC chan<- []extractedJob, baseUrl string) {
	var jobs []extractedJob
	c := make(chan extractedJob)
	pageUrl := baseUrl + "&start=" + strconv.Itoa(page*50)
	fmt.Println("Requesting : " + pageUrl)
	res, err := http.Get(pageUrl)

	checkErr(err)
	checkCode(res)

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	//searchCard := doc.Find("resultWithShelf")
	searchCard := doc.Find(".resultWithShelf")
	searchCard.Each(func(i int, card *goquery.Selection) {
		go extractJob(card, c)
		// job := extractJob(card, c)
		// jobs = append(jobs, job)
	})

	for i := 0; i < searchCard.Length(); i++ {
		job := <-c
		jobs = append(jobs, job)
	}
	mainC <- jobs
}

func extractJob(card *goquery.Selection, c chan<- extractedJob) {
	id, _ := card.Attr("data-jk")
	title := card.Find(".jobTitle").Text()
	location := card.Find(".companyLocation").Text()
	salray := card.Find(".salary-snippet").Text()
	summary := CleanString(card.Find(".job-snippet").Text())
	//fmt.Println(id, title, location, salray, summary)
	c <- extractedJob{
		id:       id,
		title:    title,
		location: location,
		salary:   salray,
		summary:  summary,
	}
}

func CleanString(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}

func getPagesNumber(baseUrl string) int {
	pages := 0
	res, err := http.Get(baseUrl)
	checkErr(err)
	checkCode(res)

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	doc.Find(".pagination").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the title

		pages = s.Find("a").Length()
	})
	return pages
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func checkCode(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalln("Request failed in stauts", res.StatusCode)
	}
}
