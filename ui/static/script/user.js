if (document.readyState == 'loading') {
    window.addEventListener('load', init);
} else {
    init();
}

async function init() {
    $('body').addEventListener('click', minimizePostForm);
    $('#dialogs-cont').addEventListener('mousedown', function (e) {
        if (e.target == this) showDialog();
    });
    $('.next_news_box').addEventListener('click', getNews);
    $('#add-news-form-textarea').addEventListener('click', maximizePostForm);

    const url = window.location.href;
    const id = url.split('id')[1];

    currentUser = {
        id: 0,
        name: '',
        lastname: '',
        avatar: 0,
        description: '',
        newsCount: 0,
        newsCountLoaded: 0,
        newsPartLoaded: 0,
        newsLastId: 0
    };

    if (!id || !id.length) return;

    const haveUser = await getCurrentUserData(id);
    if (!haveUser) return;
    
    currentUser.id = id;

    if (user.id) {
        showUserPage();

        if (user.id == id) {
            showOwnerPage();
        }
    }

    if (currentUser.id) {
        await getNews();
    }

    $('#owner-profile-form_edit-page').addEventListener('click', editPageHandler);
    $('#owner-profile-form_edit-avatar').addEventListener('click', editAvatarHandler);
    $('#edit-avatar-form_file').addEventListener('change', editAvatarFileHandler);

    $('#edit-page').addEventListener('click', updateUserInfo);
    $('#delete-page').addEventListener('click', deleteUser);

    $('#edit-avatar').addEventListener('click', loadAvatar);
    $('#delete-avatar').addEventListener('click', deleteAvatar);

    const forms = document.querySelectorAll('form');
    for (let i of forms) {
        i.addEventListener('submit', function (e) { e.preventDefault(); });
    }
}

async function editAvatarFileHandler() {
    if (!$('#edit-avatar-form_file').value) return false;

    const form = $('#edit-avatar-form');
    const formData = new FormData(form);

    try {
        const response = await fetch('/api/user/avatar', {
            method: 'POST',
            body: formData,
        });

        if (response.status == 200) {
            const resJSON = await response.json();
            currentUser.avatar = resJSON.avatar;

            $('#profile-avatar').src = getUserAvatarLink(currentUser.avatar, 'd');

            // Change avatar
        } else {
            console.error(response.status);
        }
    } catch (error) {
        // Show error
        console.error('err: ', error.message);
    }

    $('#edit-avatar-form_file').value = '';
    showDialog();
}

function editAvatarHandler() {
    if (currentUser.avatar) {
        $('#delete-avatar').classList.toggle('hidden', false);
    } else {
        $('#delete-avatar').classList.toggle('hidden', true);
    }
    showDialog('user-avatar-change');
}

function loadAvatar() {
    $('#edit-avatar-form_file').click();
}

async function deleteAvatar() {
    try {
        const response = await fetch('/api/user/avatar', {
            method: 'DELETE'
        });

        if (response.status == 200) {
            const resJSON = await response.json();
            currentUser.avatar = resJSON.avatar;

            $('#profile-avatar').src = getUserAvatarLink(currentUser.avatar, 'd');
            showDialog();

            // Change avatar in news and comms
        } else {
            console.error(response.status);
        }
    } catch (error) {
        // Show error
        console.error('err:', error.message);
    }
}

function editPageHandler() {
    const textArea = $('#input-user-info');
    if (!textArea) return false;
    textArea.value = currentUser.description;

    showDialog('user-info-change');
}

async function updateUserInfo() {
    const textArea = $('#input-user-info');
    if (!textArea) return false;

    const newsText = textArea.value;
    if (!newsText.length) return false;

    const bodyJSON = {
        description: newsText
    };

    try {
        const response = await fetch(`/api/user`, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(bodyJSON)
        });

        if (response.status == 200) {
            const resJSON = await response.json();
            if (resJSON.description) currentUser.description = resJSON.description;

            $('#profile-right_info').innerHTML = getUserInfo(currentUser.description);

            // const newsCommentLink = newsContainer.querySelector('.comment_link');
            // newsCommentLink.innerHTML = `Comments (${commentsContainer.dataset.loaded})`;

        } else {
            console.error(response.status);
        }
    } catch (error) {
        console.error('err: ', error.message);
    }

    textArea.value = '';
    showDialog();
}

async function deleteUser() {
    try {
        await fetch(`/api/user`, {
            method: 'DELETE',
        });

        route('/sign', 'replace');
    } catch (error) {
        console.error('err: ', error.message);
    }
}

function getUserInfo(info) {
    return info || 'No user information';
}

async function getCurrentUserData(id) {
    if (!id) return {};

    try {
        const response = await fetch(`/api/users/${id}`);

        if (response.status == 200) {
            const json = await response.json();

            currentUser.name = json.name;
            currentUser.lastname = json.lastname;
            if (json.avatar) currentUser.avatar = json.avatar;
            if (json.newsCount) currentUser.newsCount = json.newsCount;
            if (json.description) currentUser.description = json.description;

            const username = json.name + ' ' + json.lastname;
            $('#head_username').innerHTML = username;
            $('#profile-right_username').innerHTML = username;
            document.title = `Blog | ${username}`;

            $('#profile-avatar').src = getUserAvatarLink(currentUser.avatar, 'd');
            $('#profile-right_info').innerHTML = getUserInfo(currentUser.description);

            if (currentUser.newsCount) {
                $('#lenta-box').innerHTML = `${currentUser.newsCount} entries`;
            }

            return true;

        } else {
            console.error(response.status);
            route('/404', 'none');
            return false;
        }
    } catch (error) {
        console.error('err: ', error.message);
        route('/404', 'none');
        return false;
    }
}

async function addNews() {
    const form = $('#add-news-form');
    const textArea = form.querySelector('#add-news-form-textarea');
    const fileImage = form.querySelector('#mfile');
    const newsText = textArea.value;

    // Text and Img check
    if (!newsText.length && !fileImage.value.length) return false;

    const formData = new FormData(form);

    try {
        const response = await fetch('/api/news', {
            method: 'POST',
            body: formData,
        });

        if (response.status == 200) {
            const resJSON = await response.json();

            insertNews(resJSON, 'before');

            if (!currentUser.newsCount) {
                const createBox = $('#create_news_box');
                createBox.classList.toggle('hidden', true);
            }

            currentUser.newsCount++;
            currentUser.newsCountLoaded += resJSON.length;

            $('#lenta-box').innerHTML = `${currentUser.newsCount} entries`;
        } else {
            console.error(response.status);
        }
    } catch (error) {
        console.error('err: ', error.message);
    }

    textArea.value = '';
    $('#mfile').value = '';
    minimizePostForm();
}

async function deleteNews(newsId) {
    try {
        const response = await fetch(`/api/news/${newsId}`, {
            method: 'DELETE'
        });

        if (response.status == 204) {
            const newsItemDiv = $(`#news${newsId}`);
            if (newsItemDiv) {
                if (newsItemDiv.nextElementSibling) {
                    newsItemDiv.nextElementSibling.remove();
                }
                newsItemDiv.remove();
            }
            currentUser.newsCount--;

            if (!currentUser.newsCount) {
                $('#lenta-box').innerHTML = `There are no posts here yet`;

                const createBox = $('#create_news_box');
                createBox.innerHTML = 'Not posts here yet';
                createBox.classList.toggle('hidden', false);
            } else {
                $('#lenta-box').innerHTML = `${currentUser.newsCount} entries`;
            }
        } else {
            console.error(response.status);
        }
    } catch (error) {
        // Show error
        console.error('err: ', error.message);
    }
}

async function getNews() {
    const nextBox = $('#next_news_box');
    nextBox.innerHTML = 'Loading posts ···';
    nextBox.classList.toggle('hidden', false);

    try {
        const res = await fetch(`/api/news/user/${currentUser.id}?part=${currentUser.newsPartLoaded}&start=${currentUser.newsLastId}`);
        const resJSON = await res.json();

        if (res.status == 200 && resJSON.length) {
            insertNews(resJSON);

            currentUser.newsPartLoaded++;
            currentUser.newsCountLoaded += resJSON.length;
            if (!currentUser.newsLastId) currentUser.newsLastId = resJSON[0].id;
        }

        if (currentUser.newsCountLoaded == currentUser.newsCount) {
            nextBox.classList.toggle('hidden', true);

            if (!currentUser.newsCount) {
                const createBox = $('#create_news_box');
                createBox.innerHTML = 'Not posts here yet';
                createBox.classList.toggle('hidden', false);
            }
        }
    } catch (error) {
        console.error('err: ', error.message);
    }

    nextBox.innerHTML = 'Load posts';
}

function insertNews(news, position = 'after') {
    if (!news || !news.length) {
        return false;
    }

    if (position == 'before') {
        news.reverse();
    }

    const avatarLink = getUserAvatarLink(currentUser.avatar, 's');

    for (const [i, item] of news.entries()) {
        let commentsBadge = '';
        if (item.comments) {
            commentsBadge = ` (${item.comments})`;
        }
        let photoBlock = '';
        if (item.photos && item.photos.length) {
            let photosElems = '';
            let photosRadio = '';

            for (const [i, photo] of item.photos.entries()) {
                photosElems += `
					<div class="photo${i == 0 ? ' active' : ''}" data-photo="${i}" style="transform: translate(${i * 100}%, 0)">
						<img class="sim rounded" src="/uploads/${photo}.${imgFormat}" alt="posts image" onclick="lightbox(this.src);">
					</div>
				`;

                photosRadio += `
                    <div class="photo-radio${i == 0 ? ' active' : ''}" data-photo="${i}"></div>
                `;
            }

            photoBlock = `
				<div class="photo-box" data-photobox="${item.id}">
					${photosElems}

					<div class="arrow back" style="left:0.25em;display: none;">
						<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-arrow-left-short" viewBox="0 0 16 16">
							<path fill-rule="evenodd" d="M12 8a.5.5 0 0 1-.5.5H5.707l2.147 2.146a.5.5 0 0 1-.708.708l-3-3a.5.5 0 0 1 0-.708l3-3a.5.5 0 1 1 .708.708L5.707 7.5H11.5a.5.5 0 0 1 .5.5z"/>
						</svg>
					</div>
					<div class="arrow next" style="right:0.25em;display: none;">
						<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-arrow-right-short" viewBox="0 0 16 16">
							<path fill-rule="evenodd" d="M4 8a.5.5 0 0 1 .5-.5h5.793L8.146 5.354a.5.5 0 1 1 .708-.708l3 3a.5.5 0 0 1 0 .708l-3 3a.5.5 0 0 1-.708-.708L10.293 8.5H4.5A.5.5 0 0 1 4 8z"/>
						</svg>
					</div>

                    <div class="photo-box-radio" style="display: none;">${photosRadio}</div>
				</div>
			`;
        }

        let textBlock = '';
        if (item.text && item.text.length) {
            textBlock = `
				${item.text}
				<br>
			`;
        }

        let newsItem = `
			<img src="${avatarLink}" class="news-avatar" alt="news item avatar">
			<div class="news-info">
				<a href='/id${currentUser.id}' class="author"  data-link>${currentUser.name} ${currentUser.lastname}</a>
                <div class="delete_item owner${currentUser.id == user.id ? ' visible' : ''}" onclick="deleteNews(${item.id});">
                    <svg fill="currentColor"><title>delete post</title><use xlink:href="/static/img/bootstrap-icons.svg#trash3-fill" /></svg>
                </div>
				<br>
				${textBlock}
				${photoBlock}
				<p class="date_text">
					${formatDate(item.date)}
					 | <a href="javascript:void(0);" class="comment_link" onclick="showComments(${item.id});">Comments${commentsBadge}</a>
				</p>
				<br>
				<div id="news-comment-container${item.id}" class="hidden" data-comments="${item.comments}" data-loaded="0" data-part-loaded="0" data-last-id="0">
					<div id="news-comments${item.id}"></div>
                    <a href="javascript:void(0);" class="comment_link comment_load_link hidden" onclick="getComments(${item.id});">Load comments ···</a>
					<div id="news-comment-form${item.id}" class="user">
						<form onsubmit="return false;" class="comment-form">
							<textarea class="newc" placeholder="What are your thoughts?" name="text" rows="2"></textarea>
							<br>
							<input type="button" value="Comment" onclick="addComment(${item.id});">
						</form>
					</div>
				</div>
			</div>
		`;

        const newsItemDiv = document.createElement('div');
        newsItemDiv.id = `news${item.id}`;
        newsItemDiv.dataset.id = item.id;
        newsItemDiv.classList.toggle('news_item', true);
        newsItemDiv.classList.toggle('material-box', true);
        newsItemDiv.innerHTML = newsItem;

        if (item.photos && item.photos.length > 1) {
            const arrows = newsItemDiv.querySelectorAll('.arrow');

            arrows[0].addEventListener('click', photoBlockBack);
            arrows[1].addEventListener('click', photoBlockNext);

            //arrows[0].style.display = 'block';
            arrows[1].style.display = 'block';

            const photoBoxRadio = newsItemDiv.querySelector('.photo-box-radio');
            photoBoxRadio.style.display = 'block';
        }

        if (user.id) {
            showUserPage(newsItemDiv);
        }

        const newsItemSeparator = document.createElement('hr');

        const newsContainer = $('#news_container');

        if (position == 'after') {
            if (currentUser.newsCountLoaded > 0 || i > 0) {
                newsContainer.append(newsItemSeparator);
            }

            newsContainer.append(newsItemDiv);
        }
        if (position == 'before') {
            if (currentUser.newsCountLoaded > 0 || i > 0) {
                newsContainer.prepend(newsItemSeparator);
            }

            newsContainer.prepend(newsItemDiv);
        }
    }
}

function photoBoxChange(photoBoxId, photoId) {
    const photoBox = $(`.photo-box[data-photobox="${photoBoxId}"]`);

    let activePhoto = photoBox.querySelector('.photo.active');
    activePhoto.classList.remove('active');

    const photo = photoBox.querySelector('.photo[data-photo="' + photoId + '"]');
    photo.classList.add('active');

    let activeRadio = photoBox.querySelector('.photo-radio.active');
    activeRadio.classList.remove('active');

    const radio = photoBox.querySelector('.photo-radio[data-photo="' + photoId + '"]');
    radio.classList.add('active');

    for (const [i, photoElem] of photoBox.querySelectorAll('.photo').entries()) {
        photoElem.style.transform = `translate(${(i - photoId) * 100}%, 0)`;
    }
}

function photoBlockNext(e) {
    const elem = e.currentTarget;

    const photoBox = elem.parentElement;
    const activePhoto = photoBox.querySelector('.photo.active');
    const count = photoBox.querySelectorAll('.photo').length;
    const currentNum = parseInt(activePhoto.dataset.photo);

    if (currentNum + 1 == count) {
        return false;
    }

    photoBoxChange(photoBox.dataset.photobox, currentNum + 1);

    const photoBoxBack = photoBox.querySelector('.arrow.back');
    const photoBoxNext = photoBox.querySelector('.arrow.next');

    photoBoxBack.style.display = 'block';

    if (currentNum + 2 == count) {
        photoBoxNext.style.display = 'none';
    } else {
        photoBoxNext.style.display = 'block';
    }
}

function photoBlockBack(e) {
    const elem = e.currentTarget;

    const photoBox = elem.parentElement;
    const activePhoto = photoBox.querySelector('.photo.active');
    const currentNum = parseInt(activePhoto.dataset.photo);

    if (currentNum - 1 < 0) {
        return false;
    }

    photoBoxChange(photoBox.dataset.photobox, currentNum - 1);

    const photoBoxBack = photoBox.querySelector('.arrow.back');
    const photoBoxNext = photoBox.querySelector('.arrow.next');

    if (currentNum - 1 == 0) {
        photoBoxBack.style.display = 'none';
    } else {
        photoBoxBack.style.display = 'block';
    }

    photoBoxNext.style.display = 'block';
}

async function addComment(news_id) {
    if (!news_id) return false;

    const cont = $(`#news-comment-form${news_id}`);
    if (!cont) return false;

    const textArea = cont.querySelector('.newc');
    if (!textArea) return false;

    const newsText = textArea.value;
    if (!newsText.length) return false;

    const bodyJSON = {
        newsId: news_id,
        text: newsText
    };

    try {
        const response = await fetch('/api/comments', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(bodyJSON)
        });

        if (response.status == 200) {
            const resJSON = await response.json();

            insertComments(news_id, resJSON);
            //newsPackage++;
            const commentsContainer = $(`#news-comment-container${news_id}`);
            if (!commentsContainer) return;
            const newsConmmentsLoaded = parseInt(commentsContainer.dataset.loaded);

            commentsContainer.dataset.loaded = newsConmmentsLoaded + resJSON.length;

            const newsContainer = $(`#news${news_id}`);
            if (!newsContainer) return;

            const newsCommentLink = newsContainer.querySelector('.comment_link');
            newsCommentLink.innerHTML = `Comments (${commentsContainer.dataset.loaded})`;

        } else {
            console.error(response.status);
        }
    } catch (error) {
        console.error('err: ', error.message);
    }

    textArea.value = '';
}

async function getComments(news_id) {
    const commentsContainer = $(`#news-comment-container${news_id}`);
    if (!commentsContainer) return;

    const nextBox = commentsContainer.querySelector('.comment_load_link');
    nextBox.innerHTML = 'Loading comments ···';
    nextBox.classList.toggle('hidden', false);

    const commentsCount = parseInt(commentsContainer.dataset.comments);
    const commentsLoaded = parseInt(commentsContainer.dataset.loaded);
    const commentsPartLoaded = parseInt(commentsContainer.dataset.partLoaded);
    const commentsLastId = parseInt(commentsContainer.dataset.lastId);

    try {
        const res = await fetch(`/api/comments/news/${news_id}?part=${commentsPartLoaded}&start=${commentsLastId}`);
        const resJSON = await res.json();

        if (res.status == 200 && resJSON.length) {
            insertComments(news_id, resJSON);

            commentsContainer.dataset.partLoaded = commentsPartLoaded + 1;
            commentsContainer.dataset.loaded = commentsLoaded + resJSON.length;
            if (!commentsLastId) commentsContainer.dataset.lastId = resJSON[0].id;

            if (parseInt(commentsContainer.dataset.loaded) == commentsCount) {
                nextBox.classList.toggle('hidden', true);
            }
        }
    } catch (error) {
        console.error('err: ', error.message);
    }

    nextBox.innerHTML = 'Load comments';
}

async function deleteComment(commId) {
    try {
        const response = await fetch(`/api/comments/${commId}`, {
            method: 'DELETE'
        });

        if (response.status == 204) {
            const commentItemDiv = $(`#cid${commId}`);
            if (commentItemDiv) {
                if (commentItemDiv.previousElementSibling) {
                    commentItemDiv.previousElementSibling.remove();
                }
                commentItemDiv.remove();
            }
        } else {
            console.error(response.status);
        }
    } catch (error) {
        // Show error
        console.error('err: ', error.message);
    }
}

function insertComments(news_id, comments, position = 'after') {
    if (!comments || !comments.length) {
        return false;
    }
    const commentsContainer = $(`#news-comment-container${news_id}`);
    if (!commentsContainer) return false;
    const commentsContainerBody = $(`#news-comments${news_id}`);
    if (!commentsContainerBody) return false;

    if (position == 'before') {
        comments.reverse();
    }

    const newsConmmentsLoaded = parseInt(commentsContainer.dataset.loaded);

    for (const [i, item] of comments.entries()) {
        const avatarLink = getUserAvatarLink(item.userAvatar, 's');

        let commentItem = `
            <img src="${avatarLink}" class="comment-avatar" alt="comment-avatar">
            <div class="comment-info">
                <a href="/id${item.userId || user.id}" data-link>${item.userName || user.name} ${item.userLastName || user.lastname}</a>
                <div class="delete_item delete_comment owner ${(item.userId == user.id) ? 'user' : ''} ${(currentUser.id == user.id || item.userId == user.id) ? ' visible' : ''}" onclick="deleteComment(${item.id});">
                    <svg fill="currentColor"><title>delete comment</title><use xlink:href="/static/img/bootstrap-icons.svg#trash3-fill" /></svg>
                </div>
                <br>
                ${item.text}
                <br>
                <p class="date_text">${formatDate(item.date)}</p>
            </div>
		`;

        const commentItemDiv = document.createElement('div');
        commentItemDiv.id = `cid${item.id}`;
        commentItemDiv.dataset.id = item.id;
        commentItemDiv.classList.toggle('comment_item', true);
        commentItemDiv.innerHTML = commentItem;

        const commentItemSeparator = document.createElement('hr');

        if (position == 'after') {
            commentsContainerBody.append(commentItemSeparator);

            commentsContainerBody.append(commentItemDiv);
        }
        if (position == 'before') {
            commentsContainerBody.prepend(commentItemSeparator);

            commentsContainerBody.prepend(commentItemDiv);
        }
    }
}

// UI

function showComments(news_id) {
    const commentsContainer = $(`#news-comment-container${news_id}`);
    if (!commentsContainer) return;
    commentsContainer.classList.toggle('hidden');

    const newsConmmentsCount = parseInt(commentsContainer.dataset.comments);
    const newsConmmentsLoaded = parseInt(commentsContainer.dataset.loaded);
    if (newsConmmentsCount && !newsConmmentsLoaded) {
        getComments(news_id);
    }
}

function showOwnerPage(container = document) {
    const ownerElems = container.querySelectorAll('.owner');

    for (const elem of ownerElems) {
        elem.classList.toggle('visible', true);
    }

    $('#lenta-box').classList.toggle('rounded', false);
    $('#lenta-box').classList.toggle('rounded-top', true);
}

function showUserPage(container = document) {
    const userElems = container.querySelectorAll('.user');

    for (const elem of userElems) {
        elem.classList.toggle('visible', true);
    }
}

function minimizePostForm() {
    if ($('#add-news-form-textarea')) {
        $('#add-news-form-controls').style.display = "none";
        $('#add-news-form-textarea').rows = "3";
    }
}

function maximizePostForm() {
    if ($('#add-news-form-controls').style.display != "block") {
        $('#add-news-form-controls').style.display = "block";
        $('#add-news-form-textarea').rows = "5";
    }
}

let cid = 0;

function hicomm(cid) {
    if ($('#hcom' + cid).style.display == "block") {
        $('#hcom' + cid).style.display = "none";
        $('#comment-box_' + cid).innerHTML = "Показать все комментарии";
    } else {
        $('#hcom' + cid).style.display = "block";
        $('#comment-box_' + cid).innerHTML = "Скрыть комментарии";
    }
}

function rcom(x) {
    if (x.rows == "4") { x.rows = "2"; } else { x.rows = "4"; }
}

routes['id'].obj = { init: init };