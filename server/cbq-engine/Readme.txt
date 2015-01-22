To run cbq-engine as a standlone server the following four variables need to be initialized

NS_SERVER_CBAUTH_URL="http://<host:port>/_cbauth"
NS_SERVER_CBAUTH_USER="Administrator"
NS_SERVER_CBAUTH_PWD="<admin password>
NS_SERVER_CBAUTH_RPC_URL="http://<host:port>/cbauth-demo"

Edit start-cbq-engine.sh and update the values of these variables. 

export NS_SERVER_CBAUTH_URL="http://localhost:9000/_cbauth"
export NS_SERVER_CBAUTH_USER="Administrator"
export NS_SERVER_CBAUTH_PWD="asdasd"
export NS_SERVER_CBAUTH_RPC_URL="http://127.0.0.1:9000/cbauth-demo"

Then to start the engine run 
./start-cbq-engine <engine params>

e.g.

./start-cbq-engine.sh -datastore=http://localhost:9000/

