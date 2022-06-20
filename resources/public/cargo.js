'option strict';

function cartons() {
    return {
        async data() {
            const agents = await fetch('/api/agent', {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json'
                },
            })
                .catch(e => log(e));
            const agentsList = await agents.json()
                .catch(e => log(e));
            if (Array.isArray(agentsList) && agentsList.length > 0) {
                this.selectedEndpoint = agentsList[0];
            }
            return agentsList;
        },
        current: location.hash,
        addEndpoint: false,
        addContainer: false,
        newEndpoint: {
            id: null,
            name: null,
            server: null,
            username: null,
            password: null,
            key: null,
            credentialType: 'password',
            containers: [],
            web: {},
            db: {},
        },
        selectedEndpoint: {
            containers: [],
            credentials: {},
        },
        async saveEndpoint() {
            const { username, password, key, credentialType, server } = this.newEndpoint;
            const credentials = { username, password, key, credentialType, server };
            const endpointToSave = { ...this.newEndpoint, credentials };
            const response = await fetch('/api/agent', {
                method: 'POST',
                body: JSON.stringify(endpointToSave),
                headers: {
                    'Content-Type': 'application/json'
                },
            })
                .catch(e => log(e));
            const rsp = await response.json()
                .catch(e => log(e));
            this.$dispatch('toast', { success: rsp.success || false });
            this.addEndpoint = false;
            await this.data();
        },
        newContainer: {
            registry: null,
            image: null,
            username: null,
            password: null,
            port: null,
            env: [],
        },
        pushEnv() {
            this.newContainer.env.push({ key: null, value: null });
        },
        editContainer(container) {
            this.newContainer = container;
            this.addContainer = true;
        },
        async saveContainers() {
            if (!this.newContainer.id) {
                this.selectedEndpoint.containers.push(this.newContainer);
            }
            if (this.addEndpoint) {
                this.addContainer = false;
                return;
            }
            const containers = [...this.selectedEndpoint.containers];
            const response = await fetch(`/api/agent/container?agentId=${this.selectedEndpoint.id}`, {
                method: 'POST',
                body: JSON.stringify(containers),
                headers: {
                    'Content-Type': 'application/json'
                },
            })
                .catch(e => log(e));
            const rsp = await response.json()
                .catch(e => log(e));
            this.$dispatch('toast', { success: rsp.success || false });
            this.addContainer = false;
            await this.data();
        },
        async startContainer(ep, container) {
            console.log(ep.id, container.image);
            const response = await fetch(`/api/agent/container/manage?agentId=${ep.id}`, {
                method: 'POST',
                body: JSON.stringify({ image: container.image, registry: container.registry, action: 'start' }),
                headers: {
                    'Content-Type': 'application/json'
                },
            })
                .catch(e => log(e));
            const rsp = await response.json()
                .catch(e => log(e));
            this.$dispatch('toast', { success: rsp.success || false });
            this.addContainer = false;
        }
    };
}

const log = (...e) => console.log(e);