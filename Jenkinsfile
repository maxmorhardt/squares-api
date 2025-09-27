pipeline {
	agent {
		kubernetes {
			inheritFrom 'default'
			defaultContainer 'buildpack'
		}
	}

	parameters {
		string(name: 'DOCKER_VERSION', defaultValue: params.DOCKER_VERSION ?: '0.0.1', description: 'Docker image version', trim: true)
		string(name: 'HELM_VERSION', defaultValue: params.HELM_VERSION ?: '0.0.1', description: 'Helm chart version', trim: true)
	}


	environment {
		GITHUB_URL = 'https://github.com/maxmorhardt/squares-api'

		DOCKER_REGISTRY = 'docker.io'
		DOCKER_REGISTRY_FULL = "oci://${env.DOCKER_REGISTRY}"

		APP_NAME = "squares-api"
		CHART_NAME = "$APP_NAME-chart"
		NAMESPACE = "maxstash-apps"
	}

	stages {
		stage('Setup') {
			steps {
				script {
					withCredentials([file(credentialsId: 'kube-config', variable: 'KUBE_CONFIG')]) {
						checkout scmGit(
							branches: [[
								name: "$BRANCH_NAME"
							]],
							userRemoteConfigs: [[
								credentialsId: 'github',
								url: "$GITHUB_URL"
							]]
						)

						sh 'mkdir -p $WORKSPACE/.kube && cp $KUBE_CONFIG $WORKSPACE/.kube/config'
						sh 'ls -lah'

						echo "APP_NAME: $APP_NAME"
						echo "NAMESPACE: $NAMESPACE"
						echo "BRANCH: $BRANCH_NAME"
						echo "DOCKER_VERSION: $DOCKER_VERSION"
						echo "HELM_VERSION: $HELM_VERSION"
					}
				}
			}
		}

		stage('Go CI') {
			steps {
				script {
					sh """
						go version
						
						go mod download -x
						GOOS=linux GOARCH=arm64 go build -v -o squares-api ./cmd/main.go
					"""
				}
			}
		}

		stage('Docker CI') {
			steps {
				container('dind') {
					script {
						withCredentials([usernamePassword(credentialsId: 'docker', usernameVariable: 'DOCKER_USERNAME', passwordVariable: 'DOCKER_PASSWORD')]) {
							sh 'echo $DOCKER_PASSWORD | docker login -u $DOCKER_USERNAME --password-stdin'

							sh 'docker buildx build --platform linux/arm64/v8 . --tag $DOCKER_USERNAME/$APP_NAME:$DOCKER_VERSION --tag $DOCKER_USERNAME/$APP_NAME:latest'
							sh 'docker push $DOCKER_USERNAME/$APP_NAME --all-tags'
						}
					}
				}
			}
		}

		stage('Helm CI') {
			steps {
				script {
					withCredentials([usernamePassword(credentialsId: 'docker', usernameVariable: 'DOCKER_USERNAME', passwordVariable: 'DOCKER_PASSWORD')]) {
						sh '''
							cd helm

							echo "$DOCKER_PASSWORD" | helm registry login $DOCKER_REGISTRY --username $DOCKER_USERNAME --password-stdin

							helm package $APP_NAME --app-version=$DOCKER_VERSION --version=$HELM_VERSION
							helm push ./$CHART_NAME-${HELM_VERSION}.tgz $DOCKER_REGISTRY_FULL/$DOCKER_USERNAME
						'''
					}
				}
			}
		}

		stage('CD') {
			steps {
				script {
					withCredentials([usernamePassword(credentialsId: 'docker', usernameVariable: 'DOCKER_USERNAME', passwordVariable: 'DOCKER_PASSWORD')]) {
						sh """
							helm upgrade $APP_NAME $DOCKER_REGISTRY_FULL/$DOCKER_USERNAME/$CHART_NAME \
								--version $HELM_VERSION \
								--install \
								--atomic \
								--debug \
								--history-max=3 \
								--namespace $NAMESPACE \
								--set image.tag=$DOCKER_VERSION \
						"""
					}
				}
			}
		}
	}
}