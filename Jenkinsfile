pipeline {
    agent any
    environment {
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
                withCredentials([string(credentialsId: 'nvd-api-key', variable: 'NVD_API_KEY')]) {
                    sh """
                        ${OWASP_HOME}/bin/dependency-check.sh \
                        --project laptop-sales \
                        --scan ./ \
                        --format XML \
                        --format HTML \
                        --out ./reports \
                        --disableYarnAudit \
                        --disableNodeAudit \
                        --disableRetireJS \
                        --nvdApiKey \$NVD_API_KEY
                    """
                }
                dependencyCheckPublisher(
                    pattern: '**/reports/dependency-check-report.xml'
                )
            }
        }
    }
    post {
        always {
            archiveArtifacts artifacts: 'reports/*.html', allowEmptyArchive: true
        }
    }
}
