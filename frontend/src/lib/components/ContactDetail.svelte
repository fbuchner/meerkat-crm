<script lang="ts">
	import { onMount } from 'svelte';
	import { contactService, type Contact } from '$lib/services/contactService';
	import {
		Button,
		Card,
		Spinner,
		Tabs,
		TabItem,
		Badge,
		Input,
		Textarea,
		Select,
		ButtonGroup
	} from 'flowbite-svelte';
	import {
		ArrowLeftOutline,
		EditOutline,
		TrashBinOutline,
		CheckOutline,
		CloseOutline,
		PlusOutline
	} from 'flowbite-svelte-icons';
	import ProfilePicture from '$lib/components/ProfilePicture.svelte';
	import RelationshipList from '$lib/components/RelationshipList.svelte';

	// Define props using $props() rune
	let { contactId, back, edit, deleted } = $props<{
		contactId: number;
		back?: () => void;
		edit?: (contact: Contact) => void;
		deleted?: (contactId: number) => void;
	}>();

	let contact = $state<Contact | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Edit states
	let isEditing = $state<Record<string, boolean>>({});
	let editValues = $state<Record<string, any>>({});
	let isEditingName = $state(false);
	let editName = $state('');
	let newCircle = $state('');
	let showAddCircleInput = $state(false);

	// Field schema for editing
	const contactFieldSchema = [
		{ key: 'nickname', label: 'Nickname', type: 'text' },
		{
			key: 'gender',
			label: 'Gender',
			type: 'select',
			options: ['Male', 'Female', 'Other', 'Prefer not to say']
		},
		{ key: 'birthday', label: 'Birthday', type: 'date', format: 'DD.MM.YYYY' },
		{ key: 'email', label: 'Email', type: 'email' },
		{ key: 'phone', label: 'Phone', type: 'tel' },
		{ key: 'address', label: 'Address', type: 'text' },
		{ key: 'how_we_met', label: 'How We Met', type: 'textarea' },
		{ key: 'food_preference', label: 'Food Preference', type: 'text' },
		{ key: 'work_information', label: 'Work Information', type: 'text' },
		{ key: 'contact_information', label: 'Additional Information', type: 'textarea' }
	];

	onMount(async () => {
		await loadContact();
	});

	async function loadContact() {
		loading = true;
		error = null;

		try {
			contact = await contactService.getContact(contactId);
			if (contact) {
				editValues = { ...contact };
				editName = `${contact.firstname} ${contact.lastname}`;
				if (!contact.circles) {
					contact.circles = [];
				}
			}
		} catch (err) {
			console.error('Error loading contact:', err);
			error = err instanceof Error ? err.message : 'Failed to load contact details';
		} finally {
			loading = false;
		}
	}

	function getInitials(firstname: string, lastname: string): string {
		const firstInitial = firstname ? firstname.charAt(0).toUpperCase() : '';
		const lastInitial = lastname ? lastname.charAt(0).toUpperCase() : '';
		return `${firstInitial}${lastInitial}`;
	}

	function formatDate(dateString: string | undefined): string {
		if (!dateString) return '';
		const date = new Date(dateString);
		return date.toLocaleDateString();
	}

	function formatBirthday(birthday: string | undefined): string {
		if (!birthday) return '';
		const [year, month, day] = birthday.split('-');
		return `${day}.${month}.${year !== '0001' ? year : ''}`;
	}

	function startEditing(key: string) {
		isEditing[key] = true;
		if (key === 'birthday') {
			editValues[key] = formatBirthday(contact?.[key as keyof Contact] as string);
		} else {
			editValues[key] = contact?.[key as keyof Contact] || '';
		}
	}

	async function saveEdit(key: string) {
		if (!contact) return;

		try {
			let valueToSave = editValues[key];

			// Handle birthday format conversion
			if (key === 'birthday') {
				const datePattern = /^(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])\.(\d{4})?$/;
				const match = editValues[key].match(datePattern);
				if (match) {
					const [, day, month, year] = match;
					valueToSave = `${year || '0001'}-${month}-${day}`;
				} else if (editValues[key].trim() === '') {
					valueToSave = '';
				} else {
					error = 'Invalid birthday format. Use DD.MM.YYYY or DD.MM.';
					return;
				}
			}

			await contactService.updateContact(contactId, { [key]: valueToSave });
			// Update the contact object with the new value
			(contact as any)[key] = valueToSave;
			isEditing[key] = false;
			error = null;
		} catch (err) {
			console.error('Error updating contact:', err);
			error = err instanceof Error ? err.message : 'Failed to update contact';
		}
	}

	function cancelEdit(key: string) {
		isEditing[key] = false;
		editValues[key] = contact?.[key as keyof Contact] || '';
	}

	function startEditingName() {
		isEditingName = true;
	}

	async function saveNameEdit() {
		if (!contact) return;

		try {
			const [firstname, lastname] = editName.split(' ');
			await contactService.updateContact(contactId, {
				firstname: firstname || contact.firstname,
				lastname: lastname || contact.lastname
			});

			contact.firstname = firstname || contact.firstname;
			contact.lastname = lastname || contact.lastname;
			isEditingName = false;
			error = null;
		} catch (err) {
			console.error('Error updating name:', err);
			error = err instanceof Error ? err.message : 'Failed to update name';
		}
	}

	function cancelNameEdit() {
		isEditingName = false;
		editName = `${contact?.firstname} ${contact?.lastname}`;
	}

	function toggleAddCircle() {
		showAddCircleInput = !showAddCircleInput;
		if (showAddCircleInput) {
			// Focus the input after it's rendered
			setTimeout(() => {
				const input = document.querySelector('#new-circle-input') as HTMLInputElement;
				if (input) input.focus();
			}, 0);
		}
	}

	async function addCircle() {
		if (!contact || !newCircle.trim()) return;

		try {
			const updatedCircles = [...(contact.circles || []), newCircle.trim()];
			await contactService.updateContact(contactId, { circles: updatedCircles });

			contact.circles = updatedCircles;
			newCircle = '';
			showAddCircleInput = false;
			error = null;
		} catch (err) {
			console.error('Error adding circle:', err);
			error = err instanceof Error ? err.message : 'Failed to add circle';
		}
	}

	async function removeCircle(circleToRemove: string) {
		if (!contact) return;

		try {
			const updatedCircles = (contact.circles || []).filter((c) => c !== circleToRemove);
			await contactService.updateContact(contactId, { circles: updatedCircles });

			contact.circles = updatedCircles;
			error = null;
		} catch (err) {
			console.error('Error removing circle:', err);
			error = err instanceof Error ? err.message : 'Failed to remove circle';
		}
	}

	function goBack() {
		if (back) {
			back();
		}
	}

	function editContact() {
		if (contact && edit) {
			edit(contact);
		}
	}

	function deleteContact() {
		if (contact) {
			if (confirm(`Are you sure you want to delete ${contact.firstname} ${contact.lastname}?`)) {
				const contactId = contact.ID; // Store the ID before it might become null
				contactService
					.deleteContact(contactId)
					.then(() => {
						if (deleted) {
							deleted(contactId);
						}
						goBack();
					})
					.catch((err) => {
						error = err instanceof Error ? err.message : 'Failed to delete contact';
					});
			}
		}
	}
</script>

<div class="p-4">
	<div class="mb-6 flex items-center gap-4">
		<Button color="light" class="p-2" onclick={goBack}>
			<ArrowLeftOutline class="h-5 w-5" />
		</Button>
		<h1 class="text-3xl font-bold">Contact Details</h1>
	</div>

	{#if loading}
		<div class="my-8 flex justify-center">
			<Spinner size="12" />
		</div>
	{:else if error}
		<div
			class="mb-4 rounded-lg bg-red-100 p-4 text-sm text-red-700 dark:bg-red-200 dark:text-red-800"
		>
			<span class="font-medium">Error!</span>
			{error}
		</div>
	{:else if contact}
		<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
			<!-- Profile card -->
			<Card class="lg:col-span-1">
				<div class="flex flex-col items-center text-center">
					<ProfilePicture
						contactId={contact.ID}
						initials={getInitials(contact.firstname, contact.lastname)}
						size="xl"
						styleclass="mb-4"
					/>

					<!-- Editable Name -->
					<div class="mb-2 flex items-center gap-2">
						{#if !isEditingName}
							<h2 class="text-2xl font-bold text-gray-900 dark:text-white">
								{contact.firstname}
								{contact.lastname}
							</h2>
							<button
								class="rounded p-1 opacity-0 transition-opacity hover:bg-gray-100 hover:opacity-100 group-hover:opacity-100 dark:hover:bg-gray-700"
								onclick={startEditingName}
							>
								<EditOutline class="h-4 w-4 text-gray-500" />
							</button>
						{:else}
							<div class="flex items-center gap-2">
								<Input
									bind:value={editName}
									class="text-center text-xl font-bold"
									placeholder="First Last"
								/>
								<ButtonGroup>
									<Button size="xs" color="green" onclick={saveNameEdit}>
										<CheckOutline class="h-3 w-3" />
									</Button>
									<Button size="xs" color="red" onclick={cancelNameEdit}>
										<CloseOutline class="h-3 w-3" />
									</Button>
								</ButtonGroup>
							</div>
						{/if}
					</div>

					{#if contact.nickname}
						<p class="text-gray-600 dark:text-gray-400">"{contact.nickname}"</p>
					{/if}

					<!-- Editable Circles -->
					<div class="mt-3">
						{#if contact.circles && contact.circles.length > 0}
							<div class="mb-2 flex flex-wrap justify-center gap-2">
								{#each contact.circles as circle}
									<Badge color="blue" dismissable onclick={() => removeCircle(circle)}>
										{circle}
									</Badge>
								{/each}
							</div>
						{/if}

						<!-- Add Circle Button -->
						{#if !showAddCircleInput}
							<button
								class="inline-flex items-center gap-1 text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
								onclick={toggleAddCircle}
							>
								<PlusOutline class="h-4 w-4" />
								Add Circle
							</button>
						{:else}
							<div class="mt-2 flex items-center gap-2">
								<Input
									id="new-circle-input"
									bind:value={newCircle}
									placeholder="Circle name"
									size="sm"
									onkeydown={(e) => {
										if (e.key === 'Enter') addCircle();
										if (e.key === 'Escape') showAddCircleInput = false;
									}}
								/>
								<ButtonGroup>
									<Button size="xs" color="green" onclick={addCircle}>
										<CheckOutline class="h-3 w-3" />
									</Button>
									<Button size="xs" color="red" onclick={() => (showAddCircleInput = false)}>
										<CloseOutline class="h-3 w-3" />
									</Button>
								</ButtonGroup>
							</div>
						{/if}
					</div>

					<div class="mt-6 flex gap-2">
						<Button color="red" onclick={deleteContact}>
							<TrashBinOutline class="mr-2 h-4 w-4" />
							Delete
						</Button>
					</div>
				</div>
			</Card>

			<!-- Details tabs -->
			<Card class="lg:col-span-2">
				<Tabs style="underline">
					<TabItem open title="Details">
						<div class="space-y-4">
							{#each contactFieldSchema as field}
								{@const fieldValue = contact[field.key as keyof Contact]}
								{@const displayValue =
									field.key === 'birthday' ? formatBirthday(fieldValue as string) : fieldValue}

								<div class="group">
									<h3 class="mb-1 text-sm font-medium text-gray-500 dark:text-gray-400">
										{field.label}
									</h3>

									{#if !isEditing[field.key]}
										<div class="flex items-center gap-2">
											<p class="flex-1 text-base text-gray-900 dark:text-white">
												{displayValue || 'Not set'}
											</p>
											<button
												class="rounded p-1 opacity-0 transition-opacity hover:bg-gray-100 hover:opacity-100 group-hover:opacity-100 dark:hover:bg-gray-700"
												onclick={() => startEditing(field.key)}
											>
												<EditOutline class="h-4 w-4 text-gray-500" />
											</button>
										</div>
									{:else}
										<div class="flex items-center gap-2">
											{#if field.type === 'select'}
												<Select
													bind:value={editValues[field.key]}
													items={field.options?.map((opt) => ({ value: opt, name: opt })) || []}
													class="flex-1"
													placeholder="Select {field.label.toLowerCase()}"
												/>
											{:else if field.type === 'textarea'}
												<Textarea
													bind:value={editValues[field.key]}
													placeholder={field.label}
													rows={3}
													class="flex-1"
												/>
											{:else}
												<Input
													bind:value={editValues[field.key]}
													type={field.type === 'date' ? 'text' : field.type}
													placeholder={field.key === 'birthday'
														? 'DD.MM.YYYY or DD.MM.'
														: field.label}
													class="flex-1"
												/>
											{/if}

											<ButtonGroup>
												<Button size="xs" color="green" onclick={() => saveEdit(field.key)}>
													<CheckOutline class="h-3 w-3" />
												</Button>
												<Button size="xs" color="red" onclick={() => cancelEdit(field.key)}>
													<CloseOutline class="h-3 w-3" />
												</Button>
											</ButtonGroup>
										</div>
									{/if}
								</div>
							{/each}
						</div>
					</TabItem>

					<TabItem title="Notes">
						{#if contact.notes && contact.notes.length > 0}
							<div class="space-y-4">
								{#each contact.notes as note}
									<Card>
										<h3 class="text-lg font-semibold">{note.title}</h3>
										<p class="text-gray-700 dark:text-gray-400">{note.content}</p>
										<p class="mt-2 text-xs text-gray-500">{formatDate(note.CreatedAt)}</p>
									</Card>
								{/each}
							</div>
						{:else}
							<p class="text-gray-500 dark:text-gray-400">No notes for this contact yet.</p>
						{/if}
					</TabItem>

					<TabItem title="Relationships">
						<RelationshipList {contactId} />
					</TabItem>

					<TabItem title="Activities">
						{#if contact.activities && contact.activities.length > 0}
							<div class="space-y-4">
								{#each contact.activities as activity}
									<Card>
										<h3 class="text-lg font-semibold">{activity.title}</h3>
										<p class="text-gray-700 dark:text-gray-400">{activity.description}</p>
										<p class="mt-2 text-xs text-gray-500">
											{formatDate(activity.date)}
										</p>
									</Card>
								{/each}
							</div>
						{:else}
							<p class="text-gray-500 dark:text-gray-400">No activities for this contact yet.</p>
						{/if}
					</TabItem>
				</Tabs>
			</Card>
		</div>
	{/if}
</div>
