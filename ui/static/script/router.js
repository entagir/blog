const routes = {
    '/': {
        action: function () {
            if (user.id) route('/id' + user.id, 'replace');
            else route('/sign', 'replace');
        }
    },
    '/sign': { action: function () { }, html: 'sign.html', title: 'Blog | Sign in' },
    '/reg': { action: function () { }, html: 'reg.html', title: 'Blog | Registration' },
    '/logout': {
        action: async function () {
            user.id = 0;
            await fetch('/api/logout');
            route('/sign', 'replace');
        }
    },
    '/404': { action: function () { }, html: '404.html', title: 'Blog | Not Found' },

    '/people': { action: function () { }, html: 'people.html', script: '/static/script/people.js', title: 'Blog | Peoples' },
    'id': {
        match: 'id[0-9]+',
        html: 'user.html',
        script: '/static/script/user.js',
        title: 'Blog'
    }
};
const matches = {
    'id[0-9]+': 'id'
};

async function route(url, history = 'push') {
    console.log('route', url, history);

    let currentRoute = routes[url];
    if (!currentRoute) {
        for (const match of Object.keys(matches)) {
            if (!url.match(match)) continue;

            currentRoute = routes[matches[match]];
            break;
        }

        if (!currentRoute) return route('/', history);
    }

    if (currentRoute.title) {
        document.title = currentRoute.title;
    }

    if (history == 'push') {
        window.history.pushState({}, null, url);
    } else if (history == 'replace') {
        window.history.replaceState({}, null, url);
    }

    await loadHTML(currentRoute);

    if (typeof (currentRoute.action) == 'function') {
        currentRoute.action();
    }
}

async function loadHTML(route) {
    if (!route.html) return;

    const appBody = $('#page');
    if (!appBody) return;

    if (!route.cache) {
        const req = await fetch(`/static/pages/${route.html}`);
        const routeText = await req.text();

        route.cache = routeText;
    }

    appBody.innerHTML = route.cache;

    if (route.script) {
        if (!route.obj) {
            const scriptElem = document.createElement('script');
            scriptElem.src = route.script;
            appBody.append(scriptElem);
        } else {
            if (typeof (route.obj.init) == 'function') route.obj.init();
        }
    }
}

function initRouter() {
    document.body.addEventListener('click', function (e) {
        if (e.target.matches('[data-link]')) {
            e.preventDefault();
            const url = new URL(e.target.href).pathname;

            if (url == window.location.pathname) return;

            route(new URL(e.target.href).pathname);
        }
    });

    route(window.location.pathname);

    window.addEventListener('popstate', function (e) {
        route(window.location.pathname, 'none');
    });
}