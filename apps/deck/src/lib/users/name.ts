/** The two names of a person, joined for display. Falls back when the record holds neither. */
export function personName(person: { first_name?: string; last_name?: string }): string {
	return [person.first_name, person.last_name].filter(Boolean).join(' ') || 'Unnamed person';
}
