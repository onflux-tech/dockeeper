function escapeFilterValue(value: string): string {
	return value.replace(/\\/g, "\\\\").replace(/'/g, "\\'");
}

export function eqFilter(field: string, value?: string | null): string {
	if (!value) return "";
	return `${field} = '${escapeFilterValue(value)}'`;
}

export function containsFilter(field: string, value?: string | null): string {
	if (!value) return "";
	return `${field} ~ '${escapeFilterValue(value)}'`;
}

export function containsAnyFilter(fields: string[], value?: string | null): string {
	if (!value) return "";
	const filters = fields.map((field) => containsFilter(field, value)).filter(Boolean);
	if (filters.length === 0) return "";
	if (filters.length === 1) return filters[0];
	return `(${filters.join(" || ")})`;
}

export function andFilters(filters: Array<string | undefined | null | false>): string {
	const valid = filters.filter(
		(filter): filter is string => typeof filter === "string" && filter.length > 0,
	);
	if (valid.length === 0) return "";
	if (valid.length === 1) return valid[0];
	return valid.map((filter) => `(${filter})`).join(" && ");
}
