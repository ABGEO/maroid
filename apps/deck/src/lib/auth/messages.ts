/** The message that the shell shows for each reason the hub reports at the target of a flow. */
const messages: Record<string, string> = {
	no_identity:
		'This account is not connected to Maroid. Ask the owner of the instance for an invitation.',
	access_denied: 'This account is blocked. Ask the owner of the instance.',
	identity_taken: 'This account already belongs to another user.',
	invitation_invalid: 'This invitation is no longer valid. Ask the owner for a new one.',
	auth_failed: 'Authentication failed. Unable to sign in. Please try again.'
};

export function authFailureMessage(reason: string): string {
	return messages[reason] ?? messages.auth_failed;
}
