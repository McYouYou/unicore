kubebuilder init --domain mcyou.cn --repo github.com/mcyouyou/unicore
kubebuilder create api --group unicore --version v1 --kind Deployer
kubebuilder create api --group unicore --version v1 --kind App