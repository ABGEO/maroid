<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onDestroy } from 'svelte';

	import {
		Content,
		Form,
		createForm,
		setFormContext,
		updateErrors,
		type Schema,
		type UiSchemaRoot,
		type ValidationError
	} from '@sjsf/form';
	import { createFormValidator } from '@sjsf/cfworker-validator';
	import { theme } from '@sjsf/daisyui5-theme';
	import { createFormIdBuilder } from '@sjsf/form/id-builders/modern';
	import { createFormMerger } from '@sjsf/form/mergers/modern';
	import { resolver } from '@sjsf/form/resolvers/basic';
	import { translation } from '@sjsf/form/translations/en';

	import type { SettingsValues } from '$lib/api';

	interface Props {
		schema: Schema;
		uiSchema: UiSchemaRoot;
		initialValue: SettingsValues;
		onsubmit: (value: SettingsValues) => void;
		footer: Snippet;
	}

	let { schema, uiSchema, initialValue, onsubmit, footer }: Props = $props();

	// The parent remounts this component when the loaded settings change, so the form
	// captures them once and never reads them again.
	// svelte-ignore state_referenced_locally
	const form = createForm<SettingsValues>({
		theme,
		schema,
		uiSchema,
		initialValue,
		resolver,
		translation,
		merger: createFormMerger,
		validator: createFormValidator,
		idBuilder: createFormIdBuilder,
		onSubmit: (value) => onsubmit(value)
	});

	setFormContext(form);

	onDestroy(() => {
		form.submission.abort();
		form.fieldsValidation.abort();
	});

	/** Puts the field messages of a rejected save on their own fields. */
	export function showFieldErrors(fields: Record<string, string>): void {
		const errors: ValidationError[] = Object.entries(fields).map(([key, message]) => ({
			path: [key],
			message
		}));

		updateErrors(form, errors);
	}
</script>

<Form>
	<Content />
	{@render footer()}
</Form>
