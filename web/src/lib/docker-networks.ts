type DockerNetworkLike = {
	name?: string | null;
};

const PROTECTED_DOCKER_NETWORKS = new Set(["bridge", "host", "none"]);

export function isProtectedDockerNetwork(network: DockerNetworkLike): boolean {
	const name = network.name?.trim().toLowerCase();
	return !!name && PROTECTED_DOCKER_NETWORKS.has(name);
}