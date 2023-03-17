FROM ethereumoptimism/l2geth AS l2geth

RUN apk add curl jq bash
RUN apk add --no-cache ca-certificates
RUN geth console --exec 'console.log(admin.nodeInfo.name)' --maxpeers=0 --nodiscover --dev 2>/dev/null | head -1 > /version.txt

ADD genesis_goerli.json /genesis_goerli.json

ADD entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/bin/bash", "/entrypoint.sh"]
