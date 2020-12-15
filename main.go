package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	UriEndpoint    = ""
	DefPort        = "8083"
	Images         = "images"
	InstanceTypes  = "instance_types"
	Regions        = "regions"
	Instances      = "instances"
	BaseURI        = "/"
	InstanceSearch = "instance_search"
)

type ReqBody struct {
	//Query string `json:"query"`
	//Variables map[string]interface{} `json:"variables"`
	Offset  *int   `json:"offset"`
	Limit   *int   `json:"limit"`
	Keyword string `json:"keyword"`
}

type Result struct {
	Error    interface{} `json:"errors,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

type Region struct {
	Name string `json:"name"`
}

type Instance struct {
	Name         string      `json:"name"`
	Id           string      `json:"id"`
	PowerState   string      `json:"power_state"`
	Owner        string      `json:"owner"`
	Region       string      `json:"region"`
	InstanceType string      `json:"instance_type"`
	CreatedOn    string      `json:"createdOn"`
	Description  string      `json:"description"`
	Tags         interface{} `json:"tags"`
}

type Image struct {
	Id          string      `json:"id"`
	Name        string      `json:"name"`
	Ownership   string      `json:"ownership"`
	Owner       string      `json:"owner"`
	Description string      `json:"description"`
	Registered  string      `json:"registered"`
	Tags        interface{} `json:"tags"`
}

type InstanceType struct {
	Name           string      `json:"Instance type"`
	InstanceFamily string      `json:"Instance Family"`
	Cores          interface{} `json:"Cores"`
	Vcpus          int         `json:"vCPUs"`
}

type MasterData struct {
	Regions       []Region
	Instances     []Instance
	InstanceTypes []InstanceType
	Images        []Image
}

func (m MasterData) GetQuery(key string) []interface{} {
	ret := []interface{}{}
	switch key {
	case Regions:
		for _, reg := range m.Regions {
			ret = append(ret, reg)
		}
		return ret

	case Images:
		for _, img := range m.Images {
			ret = append(ret, img)
		}
		return ret
	case InstanceTypes:
		for _, img := range m.InstanceTypes {
			ret = append(ret, img)
		}
		return ret
	case Instances:
		for _, img := range m.Instances {
			ret = append(ret, img)
		}
		return ret
	case InstanceSearch:
		for _, img := range m.Instances {
			ret = append(ret, img)
		}
		return ret
	}
	return nil
}

var masterData MasterData

func LoadFileWithData(dataKey string, v interface{}) error {

	data, err := ioutil.ReadFile(fmt.Sprint("data/", dataKey, ".json"))
	if err != nil {
		fmt.Println("File reading error for file ", dataKey, ".json ", err.Error())
		return err
	}
	fmt.Println("Contents of file ", dataKey, ".json :")
	fmt.Println(string(data))
	err = json.Unmarshal(data, v)
	if err != nil {
		fmt.Println("Following error occured while loading file ", dataKey, ".json")
		fmt.Println(err.Error())
		return err
	}
	return nil
}

func LoadMasterData() {
	regions := []Region{}
	images := []Image{}
	instanceTypes := []InstanceType{}
	instances := []Instance{}

	// Load from local
	err := LoadFileWithData("regions", &regions)
	if err != nil {
		return
	}

	err = LoadFileWithData("instances", &instances)
	if err != nil {
		return
	}

	err = LoadFileWithData("instanceTypes", &instanceTypes)
	if err != nil {
		return
	}

	err = LoadFileWithData("images", &images)
	if err != nil {
		return
	}

	masterData = MasterData{
		Regions:       regions,
		Images:        images,
		Instances:     instances,
		InstanceTypes: instanceTypes,
	}
}

func ParseVariables(reqBody ReqBody, offsetR, limitR string) (int, int, string) {
	limit := 10
	offset := 1
	searchKeyword := ""
	var er error
	if limitR != "" {
		limit, er = strconv.Atoi(limitR)
	}
	if offsetR != "" {
		offset, er = strconv.Atoi(offsetR)
	}
	if er != nil {
		fmt.Println("Error while converting limit and offset given in query params into int")
		fmt.Print(limit, " ", offset)
		limit = 10
		offset = 0
	}
	if reqBody.Limit != nil {
		limit = *reqBody.Limit
	}
	if reqBody.Offset != nil {
		offset = *reqBody.Offset
	}
	searchKeyword = reqBody.Keyword
	return limit, offset, searchKeyword
}

func CreateResult(limit int, offset int, data []interface{}, searchKeyword string, query string) Result {
	result := Result{}
	totalCount := len(data)
	trueLimit := limit
	if offset <= 0 || limit < 1 {
		result.Error = map[string]string{
			"message": fmt.Sprint("Either offset is <= 0 or limit is less than one"),
			"code":    "402",
		}
		return result
	}
	if offset > len(data) {
		result.Error = map[string]interface{}{
			"message": fmt.Sprint("Offset cannot be greater than length of data i.e., ", totalCount),
			"code":    402,
		}
		return result
	}
	if offset+limit > len(data) {
		trueLimit = len(data) - offset + 1
	}

	resultData := []interface{}{}

	//fmt.Println("offset: ", offset)
	//fmt.Println("trueLimit: ", trueLimit)
	//fmt.Println("offset+trueLimit: ", offset+trueLimit)
	for i := offset - 1; i < offset+trueLimit-1; i++ {
		resultData = append(resultData, data[i])
	}
	result.Metadata = map[string]string{
		"totalCount": strconv.Itoa(totalCount),
		"offset":     strconv.Itoa(offset),
		"limit":      strconv.Itoa(limit),
		"kind":       query,
	}
	if searchKeyword != "" {
		m := result.Metadata.(map[string]string)
		m["keyword"] = searchKeyword
		result.Metadata = m
	}
	result.Data = resultData
	return result
}

func ExecuteQuery(ctx context.Context, query string, reqBody ReqBody, offsetR, limitR string) Result {
	//fmt.Println(query)
	limit, offset, searchKeyword := ParseVariables(reqBody, offsetR, limitR)
	if query == InstanceSearch && searchKeyword != "" {
		//TODO: Take some sophisticated approach and remove this hack
		instances := masterData.Instances
		res := []interface{}{}
		for _, inst := range instances {
			if strings.Contains(inst.Name, searchKeyword) {
				res = append(res, inst)
			}
		}
		return CreateResult(limit, offset, res, searchKeyword, query)

	}
	return CreateResult(limit, offset, masterData.GetQuery(query), "", query)
	//switch query {
	//case Regions:
	//return CreateResult(limit, offset, masterData.GetQuery(query), false, Regions)
	//case Images:
	//return Result{}
	//case InstanceTypes:
	//return Result{}
	//case Instances:
	//return Result{}
	//case InstanceSearch:
	//return Result{}
	//default:
	//return Result{
	//Error: map[string]interface{}{
	//"message": fmt.Sprint("There is no route available with name: ", query),
	//"code":    404,
	//},
	//}
	//}
}

func JsonMiddleware(resourceType string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "No query data", 400)
			return
		}
		offset := r.URL.Query().Get("offset")
		limit := r.URL.Query().Get("limit")
		//fmt.Println("QueryParams")
		//fmt.Println(limit)
		//fmt.Println(offset)
		var rBody ReqBody
		err := json.NewDecoder(r.Body).Decode(&rBody)
		if err != nil {
			//http.Error(w, "Error parsing JSON request body", 400)
			fmt.Println("Error parsing JSON request body using default configurations")
			fmt.Print("Request Body: ")
			fmt.Println(r.Body)
		}
		statusCode := 200
		fmt.Print("Route: ")
		fmt.Println(resourceType)
		fmt.Println("offset: ", offset)
		fmt.Println("length: ", limit)
		fmt.Println("Request Body: ")
		b1, _ := json.Marshal(rBody)
		fmt.Println(string(b1))
		result := ExecuteQuery(r.Context(), resourceType, rBody, offset, limit)
		if result.Error != nil {
			fmt.Printf("Failed to process the request due to:\n %+v\n", result.Error)
			statusCode = 500

		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		b, _ := json.Marshal(result)
		fmt.Fprintf(w, "%s", b)

	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = DefPort
	}

	fmt.Println("Starting server at port ", port)
	LoadMasterData()

	http.HandleFunc(fmt.Sprint(BaseURI, Images), JsonMiddleware(Images))
	http.HandleFunc(fmt.Sprint(BaseURI, InstanceTypes), JsonMiddleware(InstanceTypes))
	http.HandleFunc(fmt.Sprint(BaseURI, Regions), JsonMiddleware(Regions))
	http.HandleFunc(fmt.Sprint(BaseURI, Instances), JsonMiddleware(Instances))

	http.HandleFunc(fmt.Sprint(BaseURI, InstanceSearch), JsonMiddleware(InstanceSearch))
	http.ListenAndServe(fmt.Sprint(":", port), nil)

	fmt.Println("Server is serving on port ", port)
}
