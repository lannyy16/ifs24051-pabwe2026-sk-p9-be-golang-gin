pipeline {
 agent any
 options { timestamps(); skipDefaultCheckout(true); skipStagesAfterUnstable() }
 stages {
  stage('Checkout') { agent { docker { image 'alpine:3.21'; reuseNode true } } steps { checkout scm } }
  stage('Install Dependencies') {
   agent { docker { image 'golang:1.27-alpine'; reuseNode true; args '-e HOME=/tmp -e GOCACHE=/go-cache/build -e GOMODCACHE=/go-cache/mod -v cicd-go-cache:/go-cache' } }
   steps { sh 'set -e; go version; go mod download; go mod verify' }
  }
  stage('Test') {
   agent { docker { image 'golang:1.27-alpine'; reuseNode true; args '-e HOME=/tmp -e GOCACHE=/go-cache/build -e GOMODCACHE=/go-cache/mod -v cicd-go-cache:/go-cache' } }
   steps { sh 'set -e; go test -v -coverprofile=coverage.out -covermode=atomic ./...; go tool cover -func=coverage.out' }
   post { always { archiveArtifacts artifacts: 'coverage.out', allowEmptyArchive: true } }
  }
  stage('Build Application') {
   agent { docker { image 'golang:1.27-alpine'; reuseNode true; args '-e HOME=/tmp -e GOCACHE=/go-cache/build -e GOMODCACHE=/go-cache/mod -v cicd-go-cache:/go-cache' } }
   steps { sh 'set -e; CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server; ls -lh server' }
  }
  stage('Trivy Security Scan') {
   agent { docker { image 'aquasec/trivy:0.74.0'; reuseNode true; args '--entrypoint="" -e HOME=/tmp -e XDG_CACHE_HOME=/tmp/.cache' } }
   steps { sh 'mkdir -p .trivy-cache; trivy fs --cache-dir .trivy-cache --scanners vuln --severity HIGH,CRITICAL --format sarif --output trivy-results.sarif --exit-code 1 .' }
   post { always { recordIssues(enabledForFailure: true, failOnError: false, tools: [sarif(id: 'trivy', name: 'Trivy Security', pattern: 'trivy-results.sarif')]) } }
  }
  stage('Package Application') {
   agent { docker { image 'alpine:3.21'; reuseNode true; args '-u root' } }
   steps { sh '''
    set -e
    apk add --no-cache git zip unzip
    rm -f latest-app.zip
    git config --global --add safe.directory '*' || true
    if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then git archive --format=zip HEAD -o latest-app.zip
    else zip -r latest-app.zip . -x ".git/*" -x ".env*" -x "server" -x "*.out" -x "*.test" -x ".trivy-cache/*" -x "trivy-results.sarif" -x "latest-app.zip"; fi
    unzip -Z1 latest-app.zip | grep -E '^\.env$' && exit 1 || true
    unzip -Z1 latest-app.zip | grep -E '^server$' && exit 1 || true
    ls -lh latest-app.zip
   ''' }
  }
  stage('Publish Application') {
   steps {
    archiveArtifacts artifacts: 'latest-app.zip', fingerprint: true, allowEmptyArchive: false
    script {
     def appName=env.JOB_NAME.replaceAll('[^a-zA-Z0-9._-]','-').replaceAll('-+','-').replaceAll('^-|-$','')
     def buildId=env.BUILD_NUMBER
     sh "docker exec cicd-jenkins mkdir -p /var/jenkins_home/userContent/applications/${appName}/${buildId}; docker cp latest-app.zip cicd-jenkins:/var/jenkins_home/userContent/applications/${appName}/${buildId}/latest-app.zip"
     def base=env.BUILD_URL.substring(0,env.BUILD_URL.indexOf('/job/')).replace('localhost','host.docker.internal')
     env.ARTIFACT_URL="${base}/userContent/applications/${appName}/${buildId}/latest-app.zip"
     echo "Artifact URL: ${env.ARTIFACT_URL}"
    }
   }
  }
  stage('Deploy Application') {
   agent { docker { image 'curlimages/curl:8.15.0'; reuseNode true; args '--network cicd-network' } }
   steps {
    script {
     def reply=sh(script:'curl -sS --fail-with-body -X POST "$URL_REDEPLOY" -H "Content-Type: application/json" -d "{\\"token_access\\":\\"$DEPLOY_TOKEN\\",\\"website_id\\":\\"$WEBSITE_ID\\",\\"source_url\\":\\"$ARTIFACT_URL\\",\\"source_type\\":\\"jenkins\\"}"',returnStdout:true).trim()
     echo "Redeploy: ${reply}"
     def status='IN_PROGRESS'
     for(int i=0;i<120 && status=='IN_PROGRESS';i++){
      sleep time:5,unit:'SECONDS'
      def raw=sh(script:'curl -sS --fail-with-body -X POST "$URL_PROGRESS" -H "Content-Type: application/json" -d "{\\"token_access\\":\\"$DEPLOY_TOKEN\\",\\"website_id\\":\\"$WEBSITE_ID\\"}"',returnStdout:true).trim()
      def data=readJSON text:raw
      status=data?.data?.status?.toString()?.toUpperCase()
      if(!status) error('Response tidak memiliki data.status')
      if(status=='FAIL') error(data?.data?.log ?: 'Deployment gagal')
     }
     if(status!='SUCCESS') error("Deployment timeout: ${status}")
     echo 'DEPLOYMENT SUCCESS'
    }
   }
  }
 }
 post {
  always { archiveArtifacts artifacts:'trivy-results.sarif',allowEmptyArchive:true }
  success { echo "PIPELINE SUCCESS: ${env.ARTIFACT_URL ?: '(not published)'}" }
  failure { echo 'PIPELINE FAILED - periksa Console Output.' }
  unstable { echo 'PIPELINE UNSTABLE' }
 }
}