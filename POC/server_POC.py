from wsgiref.simple_server import make_server

def simple_app(environ, start_response):
    status = '200 OK'
    headers = [('Content-type', 'application/json'), ("Server", "C2-EX-MACHINA")]
    start_response(status, headers)
    return (b'{"NextRequestTime":1466607345,"Tasks":[{"Type":"COMMAND","Data":"ps faux && uname -a && sleep 10","Id":0},{"Type":"DOWNLOAD","Filename":"test.txt","Data":"my content","Id":1},{"Type":"UPLOAD","Data":"test.txt","Id":2}]}',)

with make_server('127.0.0.1', 8000, simple_app) as httpd:
    print("Listening on port 8000 (http://127.0.0.1:8000/)....")
    httpd.serve_forever()
