#!bin/bash
TOKEN=$1
BRANCH=$2
PULLREQ=$3

echo "START DEPLOY"
case "$BRANCH" in
"master" )
    if [[ $PULLREQ == 'false' ]]; then
        curl -H "Content-Type: application/json" -X POST -d '{"token":"'"$TOKEN"'", "branch":"'"$BRANCH"'"}' http://jaggernaut.ca:9000/hooks/deploy-respecbot-webhook
    else
        echo "Not Deploying a pull request"
    fi
    ;;
"staging" )
    if [[ $PULLREQ == 'false' ]]; then
        curl -H "Content-Type: application/json" -X POST -d '{"token":"'"$TOKEN"'", "branch":"'"$BRANCH"'"}' http://jaggernaut.ca:9000/hooks/deploy-respecbot-webhook
    else
        echo "Not Deploying a pull request"
    fi
    ;;
* )
    echo "Not a staging branch"
    ;;
esac
echo "DEPLOY OVER"