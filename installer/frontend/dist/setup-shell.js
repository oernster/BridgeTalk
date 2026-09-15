const $ = (id) => document.getElementById(id)

// appName is the product's name, read from the setup program rather than
// written here. The page renders before the name arrives, so the static text
// carries none of it and every screen fills it in from this. Written down here
// it went stale through a rename without a word from anything.
let appName = ''

// currentState is the last reading of the machine, kept so a screen that goes
// back can return to the one that was due.
let currentState = null

function backend() {
    return window.go && window.go.main && window.go.main.App
}

/* ------------------------------------------------------------------ theme */

// applyTheme sets the theme and points the button at the theme it would switch to,
// which is the same way round the application's own toggle reads. The two icons
// are the application's own artwork, copied in beside this page by genicons.py.
function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme)
    $('theme-icon').src = theme === 'dark' ? 'light-mode.png' : 'dark-mode.png'
    $('theme').title = theme === 'dark' ? 'Switch to light' : 'Switch to dark'
}

function currentTheme() {
    return document.documentElement.getAttribute('data-theme') === 'dark' ? 'dark' : 'light'
}

$('theme').onclick = () => applyTheme(currentTheme() === 'dark' ? 'light' : 'dark')

/* ---------------------------------------------------------------- screens */

function showScreen(name) {
    document.querySelectorAll('.screen').forEach((el) => el.classList.remove('active'))
    $('screen-' + name).classList.add('active')
}

function setFooter(buttons) {
    const footer = $('footer')
    footer.innerHTML = ''
    buttons.forEach((spec) => {
        const el = document.createElement('button')
        el.className = 'btn' + (spec.kind ? ' ' + spec.kind : '')
        el.textContent = spec.label
        el.onclick = spec.onClick
        if (spec.lead || spec.kind === 'primary') el.dataset.lead = 'true'
        footer.appendChild(el)
    })
    focusFooter()
}

// keyboardSettleMs is how long the webview is given to come up holding the
// keyboard before the page decides it has not.
const keyboardSettleMs = 400

// focusFooter puts focus on the button a screen leads with, so Enter does the
// obvious thing and the ring says where it would land (FR-808). That is the
// primary button unless a screen marks another as its lead, which it does where
// the action wears another colour: Uninstall is drawn as a danger yet is what
// its screen is for.
function focusFooter() {
    const footer = $('footer')
    const first = footer.querySelector('.btn[data-lead]') || footer.querySelector('.btn')
    if (first) first.focus()
}

// settleKeyboard repairs a launch that came up with no keyboard at all.
//
// Wails hands the webview its keyboard from the main window's WM_SETFOCUS, which
// Windows raises only on a change of focus; the handler for it is bound inside
// an asynchronous callback: whether the first focus arrives before there is a
// handler for it is a race. Setup loses it on a cold launch. Measured: the primary
// button was focused by this page yet drew no ring and Enter did nothing, because
// focusing an element is not the same as the document HAVING focus. The page is
// the only thing that can tell, so it checks once and asks for the keyboard back.
function settleKeyboard() {
    window.focus()
    focusFooter()
    window.setTimeout(() => {
        if (document.hasFocus()) return
        void backend().TakeKeyboard().then(() => {
            window.focus()
            focusFooter()
        })
    }, keyboardSettleMs)
}

/* ---------------------------------------------------------------- options */

// renderOptions fills a container with checkboxes and returns a reader for their
// values, so no screen has to know the ids of its own boxes.
function renderOptions(container, specs) {
    container.innerHTML = ''
    const boxes = {}
    specs.forEach((spec) => {
        const label = document.createElement('label')
        label.className = 'option'
        const input = document.createElement('input')
        input.type = 'checkbox'
        input.checked = !!spec.checked
        if (spec.onChange) input.onchange = () => spec.onChange(input.checked, input)
        const tick = document.createElement('span')
        tick.className = 'check'
        const text = document.createElement('span')
        const title = document.createElement('span')
        title.className = 'label'
        title.textContent = spec.label
        text.appendChild(title)
        if (spec.hint) {
            const hint = document.createElement('span')
            hint.className = 'hint'
            hint.textContent = spec.hint
            text.appendChild(hint)
        }
        label.append(input, tick, text)
        container.appendChild(label)
        boxes[spec.key] = input
    })
    return (key) => boxes[key].checked
}

// launchOption finishes every screen that writes files. Setup's job is done once
// the application is running, so the same tick that starts it also closes setup:
// leaving a spent installer on screen asks for a second dismissal that says
// nothing. Checked by default, because that is why people run an installer.
function launchOption() { return {
    key: 'launch',
    label: `Start ${appName} and close setup when this finishes`,
    checked: true,
} }

// shortcutOptions are the two boxes every screen that offers choices carries.
function shortcutOptions(state) {
    return [
        {
            key: 'startMenu', label: 'Add a Start Menu entry',
            hint: 'Find it by typing its name in the Start Menu.',
            checked: state.installed ? state.startMenu : true,
        },
        {
            key: 'desktop', label: 'Add a Desktop shortcut',
            checked: state.installed ? state.desktop : true,
        },
        {
            key: 'boot', label: 'Start it when I sign in',
            hint: 'It waits quietly in the notification area until the game runs.',
            checked: state.launchOnBoot,
        },
    ]
}

/* ------------------------------------------------------------------- work */

function onProgress(p) {
    $('progress-fill').style.width = p.pct + '%'
    $('progress-status').textContent = p.msg
}

async function run(work, title, doneTitle, doneMsg) {
    $('progress-title').textContent = title
    $('progress-fill').style.width = '0'
    $('progress-status').textContent = 'Starting...'
    setFooter([])
    showScreen('progress')
    try {
        await work()
        $('done-title').textContent = doneTitle
        $('done-msg').textContent = doneMsg
        showScreen('done')
        setFooter([{label: 'Close', kind: 'primary', onClick: () => backend().Quit()}])
    } catch (e) {
        showError(String(e))
    }
}

// finish runs one piece of work, then honours the launch tick. A successful
// launch closes setup; a failed one leaves the error on screen, so setup never
// disappears having quietly failed.
function finish(work, wanted, title, doneTitle, doneMsg) {
    return withAppClosed(() => run(
        () => work().then(() => {
            if (wanted) return backend().LaunchApp().then(() => backend().Quit())
        }),
        title, doneTitle, doneMsg + (wanted ? ' It is starting now.' : ''),
    ))
}

function showError(message) {
    $('error-msg').textContent = message
    showScreen('error')
    setFooter([{label: 'Close', kind: 'primary', onClick: () => backend().Quit()}])
}

// withAppClosed runs the work once the application is not running. If it is, the
// offer to close it comes first, rather than failing later on a locked file.
async function withAppClosed(proceed) {
    if (!(await backend().AppRunning())) {
        proceed()
        return
    }
    showScreen('running')
    setFooter([
        {label: 'Cancel', onClick: () => route(currentState)},
        {
            label: 'Close it and continue', kind: 'primary', onClick: async () => {
                setFooter([])
                try {
                    await backend().CloseRunningApp()
                } catch (e) {
                    showError(String(e))
                    return
                }
                proceed()
            },
        },
    ])
}

// install runs one write of the files, whatever the screen called it. The launch
// option is read here so a successful launch closes setup rather than leaving it
// waiting on a Close button nobody needs. installDir is the folder the Install screen
// chose; every other screen leaves it empty, so the write goes where the application
// already is (FR-809).
function install(read, title, doneTitle, doneMsg, installDir = '') {
    const launchAfter = read('launch')
    const choices = {
        installDir,
        startMenu: read('startMenu'),
        desktop: read('desktop'),
        launchOnBoot: read('boot'),
    }
    return withAppClosed(() => run(
        () => backend().Install(choices).then(() => {
            if (launchAfter) return backend().LaunchApp().then(() => backend().Quit())
        }),
        title, doneTitle,
        doneMsg + (launchAfter ? ' It is starting now.' : ''),
    ))
}
