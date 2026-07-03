pipeline {
    agent any
    environment {
        DOCKER_IMAGE = "fjidani/my-app"
        DOCKER_TAG = "${BUILD_NUMBER}"
        SONAR_PROJECT = "my-app"
        OWASP_HOME = "/var/jenkins_home/tools/org.jenkinsci.plugins.DependencyCheck.tools.DependencyCheckInstallation/OWASP-DC"
    }
    stages {

        stage('Git Checkout') {
            steps {
                git branch: 'main',
                    credentialsId: 'github-token',
                    url: 'https://github.com/DanielBefekadu/laptop-sales.git'
            }
        }

        stage('OWASP Dependency Check') {
            steps {
                sh """
                    ${OWASP_HOME}/bin/dependency-check.sh \
                    --scan ./ \
                    --format XML \
                    --format HTML \
                    --out ./reports \
                    --disableYarnAudit \
                    --disableNodeAudit
                """
                dependencyCheckPublisher(
                    pattern: '**/reports/dependency-check-report.xml'
                )
            }
        }

        stage('SonarQube Analysis') {
            steps {
                withSonarQubeEnv('SonarQube') {
                    sh """
                        sonar-scanner \
                        -Dsonar.projectKey=${SONAR_PROJECT} \
                        -Dsonar.projectName=${SONAR_PROJECT} \
                        -Dsonar.sources=. \
                        -Dsonar.host.url=http://10.43.17.54:9000
                    """
                }
            }
        }

        stage('Quality Gate') {
            steps {
                timeout(time: 2, unit: 'MINUTES') {
                    waitForQualityGate abortPipeline: true
                }
            }
        }

        stage('Trivy Scan') {
            steps {
                sh """
                    trivy fs \
                    --exit-code 0 \
                    --severity HIGH,CRITICAL \
                    --format table \
                    .
                """
            }
        }

        stage('Docker Build') {
            steps {
                sh "docker build -t ${DOCKER_IMAGE}:${DOCKER_TAG} ."
            }
        }

        stage('Docker Push') {
            steps {
                withCredentials([usernamePassword(
                    credentialsId: 'docker-creds',
                    usernameVariable: 'DOCKER_USER',
                    passwordVariable: 'DOCKER_PASS'
                )]) {
                    sh """
                        echo \$DOCKER_PASS | docker login -u \$DOCKER_USER --password-stdin
                        docker push ${DOCKER_IMAGE}:${DOCKER_TAG}
                        docker tag ${DOCKER_IMAGE}:${DOCKER_TAG} ${DOCKER_IMAGE}:latest
                        docker push ${DOCKER_IMAGE}:latest
                    """
                }
            }
        }

        stage('Trigger CD Pipeline') {
            steps {
                build job: 'my-app-cd',
                      wait: false,
                      parameters: [
                          string(name: 'DOCKER_TAG', value: "${DOCKER_TAG}")
                      ]
            }
        }
    }

    post {
        success {
            emailext(
                subject: "✅ Build SUCCESS: ${env.JOB_NAME} #${env.BUILD_NUMBER}",
                body: "Build passed! Check: ${env.BUILD_URL}",
                to: 'your-email@example.com'
            )
        }
        failure {
            emailext(
                subject: "❌ Build FAILED: ${env.JOB_NAME} #${env.BUILD_NUMBER}",
                body: "Build failed! Check: ${env.BUILD_URL}",
                to: 'your-email@example.com'
            )
        }
    }
}
