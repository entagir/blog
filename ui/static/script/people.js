if (document.readyState == 'loading') {
    window.addEventListener('load', init);
} else {
    init();
}

async function init() {
	await getUsers();
}

async function getUsers() {
	const nextBox = document.getElementsByClassName('next_news_box')[0];
	if (nextBox) nextBox.innerHTML = 'Loading users ···';

	try {
		const res = await fetch(`/api/users`);
		const resJSON = await res.json();

		if (res.status == 200) {
			if (!resJSON.length) {
				if (nextBox) {
					nextBox.innerHTML = 'Not users now';
					$('#no-users-alert').classList.toggle('hidden', false);
				}
				return;
			}

			insertUsers(resJSON);
		} else {
			nextBox.innerHTML = 'Load users';
		}
	} catch (error) {
		console.error('err: ', error.message);
		nextBox.innerHTML = 'Load users';
	}
}

function insertUsers(users, position='after') {
	if (!users || !users.length) {
		return false;
	}

	if (position == 'before') {
		users.reverse();
	}

	for (const [i, item] of users.entries()) {
		const avatarLink = getUserAvatarLink(item.avatar, 'd');
		let userDescription = 'No user information';
		if (item.description) {
			userDescription = item.description;
		}

		let userItem = `
			<div class="user_item">
				<img src="${avatarLink}" class="user-avatar rounded" alt="user avatar" onclick="route('/id${item.id}');">
				<div class="user-info">
					<a href="/id${item.id}" class="author" data-link>${item.name} ${item.lastname}</a>
					<p>${userDescription}</p>
				</div>
			</div>
		`;

		const tempDiv = document.createElement('div');
		tempDiv.innerHTML = userItem;

		const usersItemSeparator = document.createElement('hr');

		const usersContainer = document.getElementById('user_list');

		if (position == 'after') {
			usersContainer.append(tempDiv.firstElementChild);

			if (i < users.length - 1) {
				usersContainer.append(usersItemSeparator);
			}
		}
		if (position == 'before') {
			if (i < users.length - 1) {
				usersContainer.prepend(usersItemSeparator);
			}

			usersContainer.prepend(tempDiv.firstElementChild);
		}
	}
}

routes['/people'].obj = {init: init};