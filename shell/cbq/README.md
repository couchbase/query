# cbq

## Advanced Command Line Shell for Couchbase query.


### List of command line options : 

| Command line option | Args                  | Default Value         | Description                                                                                               | Examples                                                                                |
|---------------------|-----------------------|-----------------------|-----------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------|
| -e --engine         | <url>                 | http://localhost:8091 | URL to the query engine or couchbase cluster.                                                             | -e=http://localhost:8091 --engine=http://172.21.122.3:8093                              |
| -ne --no-engine     | --                    | false                 | Don't connect to any query service.                                                                       | --no-engine                                                                             |
| -q --quiet          | --                    | false                 | Enable/Disable startup connection message for the shell                                                   | --quiet                                                                                 |
| -t --timeout        | <val>                 | --                    | Query timeout parameter                                                                                   | --timeout=1s                                                                            |
| -u --user           | <username>            | --                    | Single username for logging into couchbase. The user will be prompted for the password if -p is not given | -u=Administrator                                                                        |
| -p --password       | <password>            | --                    | Provides the corresponding password to the username. If username not present then displays an error.      | -p=password                                                                             |
| -c --credentials    | <list of credentials> | --                    | Login Credentials. Can pass multiple credentials for SASL buckets.                                        | -c=beer-sample:password --credentials=beer-sample:password,Administrator:asdasd         |
| -v  --version       | --                    | false                 | Version of the shell                                                                                      | --version                                                                               |
| -h --help           | --                    | --                    | Help for command line options.                                                                            | --help                                                                                  |
| -s --script         | <query>               | --                    | Single command mode                                                                                       | -s="select * from `beer-sample` limit 1" --script="select * from `beer-sample` limit 1" |
| -f --file           | <input file>          | --                    | Input file to run commands from.                                                                          | -f=sample.txt --file=sample.txt                                                         |
| -o --output         | <output file>         | --                    | File to output commands and their results to.                                                             | -o=results.txt --output=results.txt                                                     |
| --pretty            | --                    | true                  | Pretty print the output.                                                                                  | --pretty=false                                                                          |
| --exit-on-error     | --                    | false                 | Exit shell on first error encountered.                                                                    | --exit-on-error                                                                         |


### List of shell commands :

| Shell Command | Args                                                            | Description                                                                                                                                                                                                                                              | Usage Example                                                                                                                   |
|---------------|-----------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------|
| \CONNECT      | <url>                                                           | Connect to the URL to the query engine or couchbase cluster.                                                                                                                                                                                             | >\CONNECT http://172.21.122.3:8093; Endpoint to Connect to : http://172.21.122.3:8093 . Type Ctrl-D / \exit / \quit to exit.    |
| \DISCONNECT   | --                                                              | Disconnect shell from the query service/ cluster endpoint.                                                                                                                                                                                               | > \DISCONNECT; Couchbase query shell not connected to any endpoint. Use \CONNECT command to connect.                            |
| \EXIT \QUIT   | --                                                              | Exit Shell                                                                                                                                                                                                                                               | > \EXIT;  $ OR > \QUIT; $                                                                                                       |
| \SET          | <parameter> <value>  <parameter> = <prefix:variable name>       | There can be 4 kinds of variables. Query parameters(-) , session variables -> User defined ($) or predefined (no prefix) and named parameters (-$). SET resets the topmost value of the stack for that variable with the given value.                    | > \SET -args,[5, "12-14-1987"]; > \SET -args [6,7];                                                                             |
| \SET          | --                                                              | Display the values for all the parameters for the current session.                                                                                                                                                                                       | >\SET;                                                                                                                          |
| \PUSH         | <parameter> <value>                                             | PUSH pushes the value onto the given parameter stack.                                                                                                                                                                                                    | >\PUSH -args [8];                                                                                                               |
| \PUSH         | --                                                              | PUSH, with no arguments, copies the top element of every variable’s stack, and then pushes that copy to the top of its respective stack. So each stack grows by 1, but the values are preserved.                                                         | >\PUSH;                                                                                                                         |
| \UNSET        | <parameter>                                                     | This deletes/resets the entire stack for that parameter. Pops the whole stack for input value and then deletes it.                                                                                                                                       | > \UNSET -args;                                                                                                                 |
| \POP          | <parameter>                                                     | Pop the top of the value stack for the given parameter.                                                                                                                                                                                                  | > \POP -timeout;                                                                                                                |
| \POP          | --                                                              | POP, with no arguments, pops every variable’s stack.                                                                                                                                                                                                     | >\POP;                                                                                                                          |
| \ALIAS        | <command name> <command>                                        | Creates a command alias for a shell command or a N1QL query. The alias is then executed using \\                                                                                                                                                         | > \ALIAS tempcommand select * from `beer-sample`; >\\tempcommand;                                                               |
| \ALIAS        | --                                                              | List all available aliases.                                                                                                                                                                                                                              | >\ALIAS; serverversion    select version()                                                                                      |
| \UNALIAS      | <alias name> ...                                                | Deletes alias.                                                                                                                                                                                                                                           | >\UNALIAS tempcommand;                                                                                                          |
| \ECHO         | <args> ...  The <args> can be parameters, aliases or any input. | Echo the value if the input is a parameter. The parameter needs to be prefixed as per its type.,If not echos the statement as is. It can also display the value of an alias command.                                                                     | > \ECHO hello \\serverversion -r; hello select version() a                                                                      |
| \VERSION      | --                                                              | Client version                                                                                                                                                                                                                                           | > \VERSION; Shell version : 1.0.0                                                                                               |
| \HELP         | <command>                                                       | Provide detailed help text for input command list.                                                                                                                                                                                                       | > \HELP VERSION;                                                                                                                |
| \HELP         | --                                                              | List all the commands supported by the shell.                                                                                                                                                                                                            | > \HELP; \HELP  \SET  \PUSH ....                                                                                                |
| \COPYRIGHT    | --                                                              | View the copyright, attributions and distribution terms.                                                                                                                                                                                                 | --                                                                                                                              |
| \SOURCE       | <filename>                                                      | Read commands from a file and execute them. The commands need to be separated by a ; and newline. For eg : temp.txt              select * from default;              \\echo this ;               ...               #this is a comment;               EOF | > \SOURCE sample.txt; create primary index on `beer-sample` using gsi; ….                                                       |
| \REDIRECT     | <filename>                                                      | Redirect the output of all the commands until \REDIRECT OFF into the file specified by filename.                                                                                                                                                         | > \REDIRECT temp_output.txt; > select * from `beer-sample`; > select abv from `beer-sample` limit 1; >\HELP; > \REDIRECT OFF; > |
| \REDIRECT OFF | --                                                              | Redirect output of subsequent commands to os.Stdout.                                                                                                                                                                                                     | >\REDIRECT OFF;                                                                                                                 |

### Parameters :

Prefix | Parameter type
-------|------
- | Query Parameter
no prefix | Predefined (Built-in) Session Variable
$ | User Defined Session Variable
-$ | Named Parameters

#### List of Predefined Parameters : histfile and auto config.
TODO :: Autoconfig will be implemented post DP.

### Error Handling
#### Connection errors (100 - 115)
	CONNECTION_REFUSED   |  100
	UNSUPPORTED_PROTOCOL |  101
	NO_SUCH_HOST         | 102
	NO_HOST_IN_URL       |  103
	UNKNOWN_PORT_TCP     |  104
	NO_ROUTE_TO_HOST     |  105
	UNREACHABLE_NETWORK  |  106
	NO_CONNECTION        |  107
	DRIVER_OPEN         |  108
	INVALID_URL          |  109

#### Read/Write/Update file errors (116 - 120)
	READ_FILE  | 116
	WRITE_FILE | 117
	FILE_OPEN  | 118
	FILE_CLOSE | 119

#### Authentication Errors (121 - 135)
Missing or invalid username/password.
	INVALID_PASSWORD   | 121
	INVALID_USERNAME   | 122
	MISSING_CREDENTIAL | 123

#### Command Errors (136 - 169)
	NO_SUCH_COMMAND | 136
	NO_SUCH_PARAM   | 137
	TOO_MANY_ARGS   | 138
	TOO_FEW_ARGS    | 139
	STACK_EMPTY     | 140
	NO_SUCH_ALIAS   | 141

#### Generic Errors (170 - 199)
	OPERATION_TIMEOUT | 170
	ROWS_SCAN         | 171
	JSON_MARSHAL      | 172
	JSON_UNMARSHAL    | 173
	DRIVER_QUERY      | 174
	WRITER_OUTPUT     | 175
	UNBALANCED_PAREN  | 176
	ROWS_CLOSE        | 177
	CMD_LINE_ARG      | 178

#### Untracked error
	UNKNOWN_ERROR | 199

#### For keyboard shortcuts see : https://github.com/peterh/liner

### Usage Examples : 
To run examples :

1. Start couchbase server on machine. Add travel-sample and beer-sample.
2. Make beer-sample a sasl bucket with password b1.
3. ./build.sh to build the shell
4. ./cbq
5. Either run the file demo.txt (\SOURCE examples/demo.txt) or manually run each command in it.














