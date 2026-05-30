const app = document.getElementById('app')!;

// --- GLOBALNY STAN KLIENTA ---
let authToken: string | null = localStorage.getItem('chess_token');
// Pobieramy zapisanego gracza z localStorage, jeśli istnieje
const savedPlayer = localStorage.getItem('chess_player'); let currentPlayer: { id: number; name: string } | null = savedPlayer ? JSON.parse(savedPlayer) : null;
const API_URL = 'http://192.168.1.101:8080/api';
let socket: WebSocket | null = null;
let activeGameId: number | null = null; // Zapamiętamy, w którą grę aktualnie gramy
let activeGames: any[] = []; // Tablica na trwające gry gracza
// // Ta zmienna przechowa nam graczy online z WebSocketa
let allPlayers: any[] = []; // Pełna baza graczy z REST (id + name)
let onlinePlayerIds: number[] = []; // Identyfikatory graczy online z WebSocketa
let diceAnimationInterval: number | null = null;
let snackbarTimeout: number | null = null;
let diceAnimElementId: string = 'bottom-dice';
let bottomDiceValue: number = 1;
let topDiceValue: number = 1;
let currentLegalMoves: number[] = [];
let selectedSquareIndex: number | null = null;
let isMyTurn: boolean = false;
let resolveDiceResponse: ((value: any) => void) | null = null;
let myColor: number | null = null;
let promotionPawnIndex: number | null = null;
let currentBudget: number = 0;

// --- HISTORIA ---
interface HistoryMoveRecord { notation: string; board: string; }
interface HalfTurn { num: number; color: number; roll: number; moves: HistoryMoveRecord[]; }
let historyData: HalfTurn[] = [];
let historyFlatMoves: Array<{ htIdx: number; mIdx: number; label: string; board: string }> = [];
let historyCurrentIdx: number = -1;
let isHistoryMode: boolean = false;
let lastGameStatePayload: any = null;
let openHistoryPanelOnLoad = false;
let pendingNavAction: 'prev' | null = null;

const PROMOTION_PIECES = [
    { type: 4, icon: '♛', name: 'Hetman', cost: 4 },
    { type: 3, icon: '♜', name: 'Wieża', cost: 3 },
    { type: 1, icon: '♝', name: 'Goniec', cost: 2 },
    { type: 2, icon: '♞', name: 'Skoczek', cost: 2 },
];
// --- 1. WIDOK LOGOWANIA ---

document.addEventListener('click', (event) => {
    const target = event.target as HTMLElement;

    // Sprawdzamy, czy kliknięty element ma ID naszego przycisku
    if (target && target.id === 'logout-btn') {
        // 1. Czyszczenie danych autoryzacyjnych (dopasuj klucze do swojego projektu)
        localStorage.removeItem('chess_token');
        localStorage.removeItem('chess_player');
        sessionStorage.clear(); // Na wypadek, gdybyś tam też coś trzymał

        switchView("login");
    }
});

function initLogin() {
    app.innerHTML = `
        <div class="screen">
            <h2>Szachy z Kostką 🎲</h2>
            <input type="text" id="username" placeholder="Wpisz swój nick..." />
            <button id="login-btn">Wejdź do gry</button>
            <div id="login-error" style="color: #ff6b6b; margin-top: 10px; font-size: 14px;"></div>
        </div>
    `;

    const loginBtn = document.getElementById('login-btn') as HTMLButtonElement;
    const usernameInput = document.getElementById('username') as HTMLInputElement;
    const errorDiv = document.getElementById('login-error')!;

    loginBtn.addEventListener('click', async () => {
        const name = usernameInput.value.trim();
        if (!name) {
            errorDiv.textContent = 'Nick nie może być pusty!';
            return;
        }

        loginBtn.disabled = true;
        loginBtn.textContent = 'Logowanie...';
        errorDiv.textContent = '';

        try {
            // Strzał do Twojego backendu w Go
            const response = await fetch(`${API_URL}/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ name: name }),
            });

            if (!response.ok) {
                throw new Error(`Serwer odpowiedział kodem: ${response.status}`);
            }

            // Oczekujemy struktury np. { token: "...", player: { id: 1, name: "Patryk" } }
            const data = await response.json();

            // Zapisujemy dane sesji
            authToken = data.token;
            currentPlayer = data.player;
            localStorage.setItem('chess_token', data.token);
            localStorage.setItem('chess_player', JSON.stringify(data.player));
            initWebSocket(); // <-- ODPALAMY WS TUTAJ
            switchView('lobby');

        } catch (error: any) {
            console.error("Błąd logowania:", error);
            errorDiv.textContent = 'Nie udało się połączyć z serwerem Go. Sprawdź czy działa i czy masz włączone CORS!';
        } finally {
            loginBtn.disabled = false;
            loginBtn.textContent = 'Wejdź do gry';
        }
    });
}

// --- 2. MOCKUP WIDOKU LOBBY (ZAKTUALIZOWANY) ---

// Interfejs odzwierciedlający strukturę gracza z Twojego serwera Go
interface Player {
    id: number;
    name: string;
}

async function fetchAllPlayers(): Promise<Player[]> {
    try {
        const response = await fetch(`${API_URL}/players`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                // Jeśli Twój endpoint wymaga tokenu, odkomentuj linijkę poniżej:
                // 'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error(`Błąd pobierania graczy: ${response.status}`);
        }
        const data = await response.json();
        return data.players;
    } catch (error) {
        console.error("Nie udało się pobrać listy graczy z REST:", error);
        // W razie błędu zwracamy pustą tablicę, żeby aplikacja się nie wywaliła
        return [];
    }
}

async function createGame(opponentId: number): Promise<void> {
    if (!authToken) {
        alert("Brak tokenu autoryzacyjnego! Zaloguj się ponownie.");
        switchView('login');
        return;
    }

    try {
        const response = await fetch(`${API_URL}/games/create`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                // Przekazujemy token autoryzacyjny zgodnie ze specyfikacją Twojego API
                'Authorization': authToken
            },
            body: JSON.stringify({
                opponent_id: opponentId
            })
        });

        if (!response.ok) {
            throw new Error(`Nie udało się utworzyć gry: ${response.status}`);
        }

        const gameData = (await response.json()).game;
        console.log("Gra utworzona pomyślnie!", gameData);

        // Wyciągamy game_id z odpowiedzi serwera (dostosuj wielkość liter ID/Id/game_id jeśli trzeba)
        const gameId = gameData.game_id || gameData.id || gameData.ID;
        joinGame(gameId);

        // TODO: W tym miejscu w przyszłości zainicjujemy połączenie WebSocket dla konkretnej gry
        // initGameWebSocket(gameId);

    } catch (error) {
        console.error("Błąd podczas tworzenia gry:", error);
        alert("Wystąpił błąd serwera przy tworzeniu gry. Sprawdź konsolę backendu.");
    }
}

function initWebSocket() {
    if (!authToken) return;

    // Jeśli połączenie już istnieje i jest otwarte, nie otwieramy nowego
    if (socket && socket.readyState === WebSocket.OPEN) return;

    // Łączymy się zgodnie ze specyfikacją Twojego API
    socket = new WebSocket(`ws://192.168.1.101:8080/ws?token=${authToken}`);

    socket.onopen = () => {
        console.log("🚀 Połączenie WebSocket zostało otwarte pomyślnie!");
    };

    socket.onmessage = (event) => {
        try {
            const message = JSON.parse(event.data);
            console.log("📥 Wiadomość z serwera (WS):", message);

            // Obsługa poszczególnych typów wiadomości z Twojego API
            switch (message.type) {
                case 'players_online':
                    handlePlayersOnline(message.payload);
                    break;
                case 'game_started':
                    handleGameStarted(message.payload);
                    break;
                case 'game_state':
                    handleGameState(message.payload);
                    break;
                case 'legal_moves':
                    currentLegalMoves = (message.payload.moves || []).map((m: any) => coordsToIndex(m));
                    handleLegalMoves(message.payload);
                    break;
                case "dice_roll":
                    if (resolveDiceResponse) {
                        resolveDiceResponse(message.payload);
                        resolveDiceResponse = null;
                    } else {
                        // Rzut przeciwnika — animujemy jego kostkę, potem aktualizujemy stan
                        const topDice = document.getElementById('top-dice');
                        if (topDice) {
                            topDice.classList.add('shaking');
                            diceAnimElementId = 'top-dice';
                            if (diceAnimationInterval) clearInterval(diceAnimationInterval);
                            diceAnimationInterval = window.setInterval(() => {
                                renderDiceDotsTo('top-dice', Math.floor(Math.random() * 6) + 1);
                            }, 70);
                        }
                        setTimeout(() => { handleGameState(message.payload); }, 1000);
                    }
                    break;
                case 'history':
                    handleHistory(message.payload);
                    break;
                case 'error':
                    showSnackbar(message.payload.error ?? 'Nieznany błąd serwera');
                    break;
                default:
                    console.warn("Nienany typ wiadomości:", message.type);
            }
        } catch (err) {
            console.error("Błąd parsowania wiadomości WebSocket:", err);
        }
    };

    socket.onclose = () => {
        console.log("🔌 Połączenie WebSocket zostało zamknięte.");
        // Opcjonalnie: auto-reconnect po kilku sekundach
    };

    socket.onerror = (error) => {
        console.error("💥 Błąd WebSocket:", error);
    };
}

async function initLobby() {
    // Pobieramy dane z obu endpointów REST jednocześnie
    const [playersData, gamesData] = await Promise.all([
        fetchAllPlayers(),
        fetchActiveGames()
    ]);

    // Go pakuje dane w obiekty .players i .games, upewniamy się że wyciągamy tablice
    allPlayers = playersData;
    activeGames = gamesData;

    app.innerHTML = `
        <div class="screen">
            <div class="lobby-header">
                <h3>Witaj w Lobby, <span id="lobby-username" style="color: #4caf50;">Gracz</span>!</h3>
                <div style="display:flex;gap:8px;">
                    <button onclick="showRulesModal()" class="rules-button">Zasady 📖</button>
                    <button id="logout-btn" class="logout-button">Wyloguj się 🚪</button>
                </div>
            </div>
            
            <div class="lobby-section">
                <h4>Gracze Online:</h4>
                <ul id="online-players-list" class="player-container"></ul>
            </div>

            <div class="lobby-section" style="margin-top: 15px;">
                <h4>Gracze Offline:</h4>
                <ul id="offline-players-list" class="player-container" style="opacity: 0.6;"></ul>
            </div>

            <hr style="width: 100%; border: 0; border-top: 1px solid #555; margin: 20px 0;" />

            <div class="lobby-section">
                <h4>Twoje trwające gry:</h4>
                <ul id="active-games-list"></ul>
            </div>
        </div>
    `;

    if (currentPlayer) {
        document.getElementById('lobby-username')!.textContent = currentPlayer.name;
    }

    // Wywołujemy renderowanie struktur
    renderLobbyLists();
}

// Funkcja pomocnicza: szuka nicku w pobranej bazie allPlayers na podstawie ID
function getPlayerNameById(id: number): string {
    const player = allPlayers.find(p => p.id === id || p.ID === id);
    return player ? (player.name || player.Name) : `Gracz #${id}`;
}
function renderLobbyLists() {
    const onlineList = document.getElementById('online-players-list');
    const offlineList = document.getElementById('offline-players-list');
    const gamesList = document.getElementById('active-games-list');

    if (!onlineList || !offlineList || !gamesList) return;

    // Filtrujemy aktualnego gracza – nie chcemy go na żadnej liście do wyzwania
    const otherPlayers = allPlayers.filter(p => {
        const pId = p.id !== undefined ? p.id : p.ID;
        return currentPlayer ? pId !== currentPlayer.id : true;
    });

    // Podział na online i offline na podstawie ID zebranych z WebSocketa
    const onlineUsers = otherPlayers.filter(p => {
        const pId = p.id !== undefined ? p.id : p.ID;
        return onlinePlayerIds.includes(pId);
    });

    const offlineUsers = otherPlayers.filter(p => {
        const pId = p.id !== undefined ? p.id : p.ID;
        return !onlinePlayerIds.includes(pId);
    });

    // 1. Renderowanie listy ONLINE
    if (onlineUsers.length === 0) {
        onlineList.innerHTML = `<li style="color: #aaa; font-style: italic; font-size: 14px;">Brak innych graczy online...</li>`;
    } else {
        onlineList.innerHTML = onlineUsers.map(p => `
            <li class="player-item" data-id="${p.id || p.ID}" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 5px;">
                <span>🟢 <b>${p.name || p.Name}</b></span>
                <button class="play-with-btn" onclick="handleChallengeClick(${p.id || p.ID})">Wyzwij</button>
            </li>
        `).join('');
    }

    // 2. Renderowanie listy OFFLINE (brak przycisku gry, bo są niedostępni)
    if (offlineUsers.length === 0) {
        offlineList.innerHTML = `<li style="color: #aaa; font-style: italic; font-size: 14px;">Wszyscy są online!</li>`;
    } else {
        offlineList.innerHTML = offlineUsers.map(p => `
            <li style="margin-bottom: 5px; color: #bbb;">
                <span>🔴 ${p.name || p.Name}</span>
                <button class="play-with-btn" onclick="handleChallengeClick(${p.id || p.ID})">Wyzwij</button>
            </li>
        `).join('');
    }

    // 3. Renderowanie trwających gier (teraz bezpośrednio, bo backend już je przefiltrował)
    if (activeGames.length === 0) {
        gamesList.innerHTML = `<li style="color: #aaa; font-style: italic; font-size: 14px;">Nie uczestniczysz obecnie w żadnej grze.</li>`;
    } else {
        gamesList.innerHTML = activeGames.map(g => {
            const gameId = g.game_id || g.id || g.ID;

            const whiteId = g.white_id || g.WhiteID || g.white_player_id;
            const blackId = g.black_id || g.BlackID || g.black_player_id;

            const whiteName = getPlayerNameById(whiteId);
            const blackName = getPlayerNameById(blackId);

            return `
                <li style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; background: #2a2a2a; padding: 6px 10px; border-radius: 4px;">
                    <span>🏆 Mecz #${gameId}: <b style="color: #fff;">${whiteName}</b> vs <b>${blackName}</b></span>
                    <button class="join-game-btn" onclick="handleJoinClick(${gameId})">Dołącz</button>
                </li>
            `;
        }).join('');
    }
}


function handleJoinClick(gameId: number) {
    joinGame(gameId);
}

// Ta funkcja zostanie wywołana przy kliknięciu przycisku "Wyzwij"
async function handleChallengeClick(opponentId: number) {
    if (!currentPlayer) return;

    // 1. BLOKADA DUPLIKATÓW: Szukamy w aktualnej liście aktywnych gier, 
    // czy już gramy z tym konkretnym użytkownikiem
    const existingGame = activeGames.find(g => {
        const whiteId = g.white_id || g.WhiteID || g.white_player_id;
        const blackId = g.black_id || g.BlackID || g.black_player_id;

        // Sprawdzamy czy to mecz Ja vs Przeciwnik lub Przeciwnik vs Ja
        return (whiteId === currentPlayer!.id && blackId === opponentId) ||
            (blackId === currentPlayer!.id && whiteId === opponentId);
    });

    if (existingGame) {
        const gameId = existingGame.game_id || existingGame.id || existingGame.ID;
        console.log(`Gra z tym użytkownikiem już trwa (Mecz #${gameId}). Automatycznie dołączam...`);
        joinGame(gameId);
        return; // Przerywamy działanie, nie wysyłamy POST-a!
    }

    // 2. Jeśli nie ma duplikatu, blokujemy przycisk (wizualnie) i tworzymy nową grę
    console.log(`Brak trwających gier z użytkownikiem ${opponentId}. Tworzę nową grę...`);

    // Pobieramy przycisk, który został kliknięty, żeby dać feedback wizualny
    const btn = document.querySelector(`button[onclick="handleChallengeClick(${opponentId})"]`) as HTMLButtonElement;
    if (btn) {
        btn.disabled = true;
        btn.textContent = 'Tworzenie...';
    }

    await createGame(opponentId);

    if (btn) {
        btn.disabled = false;
        btn.textContent = 'Wyzwij';
    }
}
// Przypisujemy funkcje do globalnego obiektu window
(window as any).handleChallengeClick = handleChallengeClick;
(window as any).handleJoinClick = handleJoinClick;

function showRulesModal() {
    let overlay = document.getElementById('rules-modal-overlay') as HTMLElement | null;
    if (overlay) { overlay.style.display = 'flex'; return; }

    overlay = document.createElement('div');
    overlay.id = 'rules-modal-overlay';
    overlay.className = 'game-modal-overlay';
    overlay.style.display = 'flex';
    overlay.style.zIndex = '1000000';

    overlay.innerHTML = `
        <div class="game-modal-content rules-modal-content">
            <button class="modal-close-btn" id="rules-close-btn">✕</button>
            <h2>Zasady Gry 📖</h2>
            <div class="rules-body">
                <p>Przed rozpoczęciem gracze rzucają kostką — wyższy wynik gra <b>białymi</b>.</p>
                <p>Rzut kostką ustala <b>środki</b> na daną turę.</p>

                <h3>Koszt ruchu</h3>
                <table class="rules-cost-table">
                    <tr><td>♟ Pion</td><td>1 pkt</td></tr>
                    <tr><td>♝ Goniec, ♞ Skoczek, ♚ Król</td><td>2 pkt</td></tr>
                    <tr><td>♜ Wieża</td><td>3 pkt</td></tr>
                    <tr><td>♛ Hetman</td><td>4 pkt</td></tr>
                    <tr><td>Roszada</td><td>2 pkt</td></tr>
                </table>

                <h3>Przebieg tury</h3>
                <ul>
                    <li>Gracz <b>musi</b> wykonać co najmniej jeden ruch, jeśli ma na to środki.</li>
                    <li>Po ruchu można zrezygnować z pozostałych środków — <b>nie przechodzą</b> do następnej tury.</li>
                    <li>Brak środków na jakikolwiek legalny ruch → gracz traci turę, a środki przechodzą do jego kolejnej tury.</li>
                    <li>Daną figurą można ruszać wielokrotnie w ramach środków, ale <b>maksymalnie do pierwszego bicia</b> tą figurą.</li>
                    <li>Po zakończeniu tury sytuacja na planszy musi być inna niż na jej początku.</li>
                </ul>

                <h3>Szach i szach-mat</h3>
                <ul>
                    <li>Jeśli Twój król jest szachowany, musisz wyjść z szacha <b>pierwszym ruchem</b>. Brak wyjścia = szach-mat.</li>
                    <li>Szachowany król kosztuje <b>1 pkt</b> (żeby zawsze można było wyjść z szacha po rzucie 1).</li>
                    <li>Królem nie można poruszać się na pole, na którym byłby szachowany.</li>
                </ul>

                <h3>Promocja piona</h3>
                <ul>
                    <li>Ruch piona na ostatnie pole kosztuje standardowo <b>1 pkt</b>.</li>
                    <li>Promocja wymaga środków na ruch figury, na którą pion jest promowany.</li>
                </ul>

                <p style="margin-top:16px;color:#797977;font-size:12px;">W sytuacjach nierozstrzygniętych powyższymi zasadami obowiązują standardowe zasady szachów.</p>
            </div>
        </div>
    `;

    document.body.appendChild(overlay);
    document.getElementById('rules-close-btn')!.addEventListener('click', hideRulesModal);
    overlay.addEventListener('click', (e) => { if (e.target === overlay) hideRulesModal(); });
}

function hideRulesModal() {
    const overlay = document.getElementById('rules-modal-overlay');
    if (overlay) overlay.style.display = 'none';
}

(window as any).showRulesModal = showRulesModal;

function joinGame(gameId: number) {
    console.log(`Próba dołączenia do gry o ID: ${gameId}`);
    activeGameId = gameId;

    // 1. Przełączamy ekran na szachownicę (Vite wyrenderuje pustą planszę)
    switchView('game');

    // 2. Informujemy serwer, że chcemy stan tej gry.

    if (socket && socket.readyState === WebSocket.OPEN) {
        console.log(`Wysyłam prośbę o synchronizację gry #${gameId}...`);

        sendWSMessage("join_game", {
            game_id: gameId
        });
    }
}

// --- 3. MOCKUP WIDOKU GRY ---
function initGame() {
    app.innerHTML = `
        <div class="game-container">
            <span id="game-turn-info">Ładowanie...</span>
            <div class="player-banner" id="top-player-banner"></div>
            <div class="board-wrapper">
                <div class="chessboard" id="board"></div>
                <button id="roll-dice-btn" class="center-dice-btn">RZUĆ KOSTKĄ 🎲</button>
            </div>
            <div class="player-banner" id="bottom-player-banner"></div>
            <div class="cost-legend">
                <span class="cost-legend-title">Koszt:</span>
                <span class="cost-item">♟ <span class="cost-value">1</span></span>
                <span class="cost-sep">·</span>
                <span class="cost-item">♝♞ <span class="cost-value">2</span></span>
                <span class="cost-sep">·</span>
                <span class="cost-item">♚ <span class="cost-value">2</span></span>
                <span class="cost-sep">·</span>
                <span class="cost-item">♜ <span class="cost-value">3</span></span>
                <span class="cost-sep">·</span>
                <span class="cost-item">♛ <span class="cost-value">4</span></span>
                <span class="cost-sep">·</span>
                <span class="cost-item">Roszada <span class="cost-value">2</span></span>
            </div>
            <div><button id="end-turn-btn">Zakończ turę</button></div>
            <div class="move-nav-row" id="move-nav-row">
                <button id="nav-prev" class="nav-arrow-btn" onclick="navHistoryPrev()">◀</button>
                <span id="nav-pos" class="nav-pos">—</span>
                <button id="nav-next" class="nav-arrow-btn" onclick="navHistoryNext()">▶</button>
                <button id="nav-latest" class="nav-arrow-btn" onclick="navHistoryLatest()" title="Aktualny ruch">▶|</button>
            </div>
            <div style="display:flex;gap:8px;">
                <button id="history-btn" onclick="openHistory()" class="rules-button">Ruchy 📜</button>
                <button onclick="showRulesModal()" class="rules-button">Zasady 📖</button>
                <button onclick="window.switchView('lobby')">Wyjdź do lobby</button>
            </div>
        </div>
    `;

    const board = document.getElementById('board')!;
    document.getElementById('end-turn-btn')?.addEventListener('click', handleEndTurn);

    // Stan gry na potrzeby makiety (wybrana pozycja startowa)

    // 1. Generowanie planszy 8x8
    for (let i = 0; i < 64; i++) {
        const row = Math.floor(i / 8);
        const col = i % 8;
        const isLight = (row + col) % 2 === 0;

        const square = document.createElement('div');
        square.classList.add('square', isLight ? 'light' : 'dark');
        square.dataset.index = i.toString();

        // Mechanizm obsługi kliknięć (alternatywa dla Drag&Drop, idealna na Mobile)
        square.addEventListener('click', () => {
            const hasPiece = square.querySelector('.piece') !== null;
            if (hasPiece && selectedSquareIndex === null) {
                if (square.dataset.promotable === 'true') handleSquareClick(i);
                return;
            }
            handleSquareClick(i);
        });

        board.appendChild(square);
    }



    // --- LOGIKA REPRODUKCJI KROPEK NA KOSTCE (BEZ ZMIAN) ---
    const rollBtn = document.getElementById('roll-dice-btn') as HTMLButtonElement;


    rollBtn.addEventListener('click', () => {
        const bottomDice = document.getElementById('bottom-dice')!;
        rollBtn.disabled = true;
        bottomDice.classList.add('shaking');
        diceAnimElementId = 'bottom-dice';

        if (diceAnimationInterval) clearInterval(diceAnimationInterval);
        diceAnimationInterval = window.setInterval(() => {
            renderDiceDotsTo('bottom-dice', Math.floor(Math.random() * 6) + 1);
        }, 70);

        handleRollDice();
    });
}

// 2. Obsługa logiki klikania (Wybór -> Ruch)
function handleSquareClick(clickedIndex: number) {
    const board = document.getElementById('board')!;
    const clickedSquare = board.querySelector(`[data-index="${clickedIndex}"]`) as HTMLElement;
    const currentSelected = board.querySelector('.selected') as HTMLElement;

    if (clickedSquare.dataset.promotable === 'true') {
        if (currentSelected) currentSelected.classList.remove('selected');
        selectedSquareIndex = null;
        clearLegalMoves();
        showPromotionDropdown(clickedIndex);
        return;
    }

    if (selectedSquareIndex === null) {
        // KROK 1: Wybór figury poprzez kliknięcie
        console.log("handleSquareClick2");
        if (clickedSquare.textContent.trim() !== "") {
            selectedSquareIndex = clickedIndex;
            clickedSquare.classList.add('selected');
        }
    } else {
        // KROK 2: Wybór pola docelowego
        if (selectedSquareIndex != clickedIndex) {
            console.log("handleSquareClick4");
            // Wykonanie ruchu w makiecie
            executeMove(selectedSquareIndex, clickedIndex);
            if (currentSelected) currentSelected.classList.remove('selected');
            clearLegalMoves();
            selectedSquareIndex = null;
        }
    }
}

function renderDiceDotsTo(id: string, value: number) {
    if (id === 'bottom-dice') bottomDiceValue = value;
    if (id === 'top-dice') topDiceValue = value;
    const el = document.getElementById(id);
    if (!el) return;
    const dotPositions: { [key: number]: number[] } = {
        1: [4], 2: [0, 8], 3: [0, 4, 8], 4: [0, 2, 6, 8], 5: [0, 2, 4, 6, 8], 6: [0, 2, 3, 5, 6, 8]
    };
    if (value === 0) value = 1;
    el.innerHTML = '';
    for (let i = 0; i < 9; i++) {
        const cell = document.createElement('div');
        if (dotPositions[value].includes(i)) {
            const dot = document.createElement('div');
            dot.classList.add('dot');
            cell.appendChild(dot);
        }
        el.appendChild(cell);
    }
}

function showSnackbar(message: string, type: 'error' | 'info' | 'success' = 'error') {
    let el = document.getElementById('snackbar') as HTMLElement | null;
    if (!el) {
        el = document.createElement('div');
        el.id = 'snackbar';
        el.className = 'snackbar';
        document.body.appendChild(el);
    }
    el.classList.remove('visible', 'error', 'info', 'success');
    el.textContent = message;
    el.classList.add(type);
    void (el as HTMLElement).offsetHeight; // reflow żeby animacja startowała od nowa
    el.classList.add('visible');
    if (snackbarTimeout) clearTimeout(snackbarTimeout);
    snackbarTimeout = window.setTimeout(() => {
        el!.classList.remove('visible');
        snackbarTimeout = null;
    }, 3000);
}

function handleEndTurn() {
    if (!activeGameId || !socket || socket.readyState !== WebSocket.OPEN) return;

    console.log(`Wysyłam żądanie zakończenia tury dla gry #${activeGameId}`);

    sendWSMessage("end_turn", {
        game_id: activeGameId
    });
}
function handleRollDice() {
    const rollBtn = document.getElementById('roll-dice-btn');
    if (rollBtn) rollBtn.style.display = 'none';
    if (!activeGameId || !socket || socket.readyState !== WebSocket.OPEN) return;

    const networkPromise = new Promise((resolve) => {
        resolveDiceResponse = resolve; // Zapisujemy "wyzwalacz" do zmiennej globalnej
    });
    const timerPromise = new Promise((resolve) => setTimeout(resolve, 1000));

    sendWSMessage("roll_dice", {
        game_id: activeGameId
    });

    Promise.all([networkPromise, timerPromise]).then(([serverResponse]) => {
        // 🔥 TEN BLOK URUCHOMI SIĘ DOPIERO PO MINIMUM 1 SEKUNDZIE
        // (i dopiero gdy serwer odeśle dane)

        //        stopDiceAnimation(); // Zatrzymujemy animację

        // Pokazujemy ostateczny wynik na kostce
        //renderDiceResult(serverResponse.CurrentDice);

        // Na samym końcu bezpiecznie aktualizujemy budżet i resztę stanu
        handleGameState(serverResponse);
    });
}

// 3. Obsługa Drag & Drop za pomocą Pointer Events (PC & Mobile)
function setupPieceDragAndDrop(piece: HTMLElement, startIndex: number) {
    piece.addEventListener('dragstart', (e) => e.preventDefault());
    // Nasłuchujemy na parentSquare (czyli na kafelku), bo figura ma pointer-events: none
    const parentSquare = piece.parentElement!;

    parentSquare.addEventListener('pointerdown', (e) => {
        // Interweniujemy tylko, jeśli na kafelku jest nasza figura
        document.querySelectorAll('.square').forEach(s => s.classList.remove('selected'));
        if (!parentSquare.contains(piece)) return;
        parentSquare.classList.add('selected');
        clearLegalMoves();
        const initialIndex = parseInt(parentSquare.dataset.index!);
        selectedSquareIndex = initialIndex;
        const startX = e.clientX;
        const startY = e.clientY;
        console.log(initialIndex);
        console.log(indexToCoords(initialIndex));
        let isDragging = false;
        const rect = piece.getBoundingClientRect();

        function onPointerMove(ev: PointerEvent) {
            if (!piece || !rect) return;

            const boardElement = document.getElementById('board');
            const isFlipped = boardElement?.classList.contains('flipped');

            const rawDeltaX = ev.clientX - startX;
            const rawDeltaY = ev.clientY - startY;
            const distance = Math.sqrt(rawDeltaX * rawDeltaX + rawDeltaY * rawDeltaY);

            if (!isDragging && distance < 7) return;

            if (!isDragging) {
                getLegalMoves(startIndex);
                isDragging = true;
                ev.preventDefault();

                piece.classList.add('dragging');
                piece.style.width = rect.width + 'px';
                piece.style.height = rect.height + 'px';
                piece.style.position = 'fixed';
                piece.style.zIndex = '1000';
            }

            if (isFlipped && boardElement) {
                const boardRect = boardElement.getBoundingClientRect();
                const flippedX = boardRect.right - ev.clientX;
                const flippedY = boardRect.bottom - ev.clientY;
                piece.style.left = (flippedX - rect.width / 2) + 'px';
                piece.style.top = (flippedY - rect.height / 2) + 'px';
            } else {
                // Standardowy widok dla białego (bez obrotu)
                piece.style.left = (ev.clientX - rect.width / 2) + 'px';
                piece.style.top = (ev.clientY - rect.height / 2) + 'px';
            }
        }

        document.addEventListener('pointermove', onPointerMove);

        const onPointerUp = function (ev: PointerEvent) {
            document.removeEventListener('pointermove', onPointerMove);
            document.removeEventListener('pointerup', onPointerUp);

            if (!isDragging) {
                getLegalMoves(startIndex);
                // Jeśli nie przesunięto o 7px, nie robimy nic. 
                // Przeglądarka sama naturalnie odpali standardowy 'click' na kafelku!
                return;
            }

            ev.preventDefault();
            ev.stopPropagation();

            // Na czas szukania pola docelowego na moment wyłączamy widoczność figury dla elementFromPoint
            piece.style.display = 'none';
            const dropTarget = document.elementFromPoint(ev.clientX, ev.clientY) as HTMLElement;
            const targetSquare = dropTarget?.closest('.square') as HTMLElement;
            piece.style.display = '';

            // Reset stylów
            piece.classList.remove('dragging');
            piece.style.position = '';
            piece.style.zIndex = '';
            piece.style.left = '';
            piece.style.top = '';
            piece.style.width = '';
            piece.style.height = '';
            parentSquare.classList.remove('selected');
            clearLegalMoves();

            if (targetSquare) {
                const targetIndex = parseInt(targetSquare.dataset.index!);
                if (initialIndex !== targetIndex) {
                    executeMove(initialIndex, targetIndex);
                }
            }
            clearLegalMoves();
        };

        document.addEventListener('pointerup', onPointerUp);
    });
}

// Pomocnicza funkcja, która zamienia indeks tablicy (0-63) na strukturę {"Row": x, "Col": y} dla Go
function indexToCoords(index: number) {
    return {
        Row: Math.floor(index / 8),
        Col: index % 8
    };
}

// Pomocnicza funkcja, która zamienia indeks tablicy (0-63) na strukturę {"Row": x, "Col": y} dla Go
function coordsToIndex(coords: any) {
    return coords.Row * 8 + coords.Col;
}

function getLegalMoves(startIndex: number) {
    if (isMyTurn) {
        sendWSMessage("get_legal_moves", {
            game_id: activeGameId,
            from: indexToCoords(startIndex)
        });
    }
}

// 4. Wspólna funkcja wykonująca fizyczne przeniesienie figury w DOM
function executeMove(fromIndex: number, toIndex: number) {
    if (!activeGameId) {
        console.error("Brak aktywnego ID gry!");
        return;
    }

    const isMoveLegal = currentLegalMoves.includes(toIndex);
    if (isMyTurn && isMoveLegal) {

        sendWSMessage("make_move", {
            game_id: activeGameId,
            from: indexToCoords(fromIndex),
            to: indexToCoords(toIndex)
        });
    }
}

function sendWSMessage(type: string, payload: any) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        console.error("Nie można wysłać wiadomości. WebSocket nie jest połączony.");
        return;
    }
    const message = JSON.stringify({ type, payload });
    socket.send(message);
    console.log("📤 Wysłano przez WS:", message);
}

async function handlePlayersOnline(payload: any) {
    console.log("Aktualizacja statusów online z WS:", payload);

    const rawList = payload.players || payload;

    if (Array.isArray(rawList)) {
        onlinePlayerIds = rawList.map(item => {
            if (typeof item === 'object' && item !== null) {
                return item.id || item.ID;
            }
            return item;
        });
    }

    // --- KLUCZOWA POPRAWKA ---
    // Sprawdzamy, czy wśród zalogowanych ID jest ktoś, kogo NIE mamy w lokalnej bazie allPlayers
    const hasUnknownPlayer = onlinePlayerIds.some(id => {
        return !allPlayers.some(p => p.id === id || p.ID === id);
    });

    // Jeśli pojawił się nowy gracz, szybko dociągamy aktualną listę z bazy przez REST
    if (hasUnknownPlayer) {
        console.log("Wykryto nowego gracza na serwerze! Aktualizuję bazę nicków przez REST...");
        const playersData = await fetchAllPlayers();
        allPlayers = playersData;
    }
    // -------------------------

    // Teraz renderowanie ma już komplet danych i nowy nick pojawi się natychmiast!
    renderLobbyLists();
}

function handleGameStarted(payload: any) {
    console.log("Wykryto start nowej gry na serwerze:", payload);

    // Ponownie dociągamy listę aktywnych gier z REST, żeby lobby się zaktualizowało
    fetchActiveGames().then(gamesData => {
        activeGames = gamesData;
        renderLobbyLists(); // Odświeżamy widok lobby u wszystkich zainteresowanych
    });
}

function hideGameOverModal() {
    const overlay = document.getElementById('game-over-overlay');
    if (overlay) overlay.style.display = 'none';
}

function handleGameOver(winnerColor: number) {
    const rollButton = document.getElementById('roll-dice-btn');
    if (rollButton) rollButton.style.display = 'none';

    const overlay = document.getElementById('game-over-overlay');
    const titleElement = document.getElementById('modal-title');
    const messageElement = document.getElementById('modal-message');
    const lobbyButton = document.getElementById('modal-lobby-btn');
    const closeButton = document.getElementById('modal-close-btn');

    if (!overlay || !titleElement || !messageElement) return;

    const winnerName = winnerColor === 0 ? "Białe" : "Czarne";

    if (winnerColor === myColor) {
        titleElement.innerText = "🎉 ZWYCIĘSTWO!";
        titleElement.style.color = "#ffd700";
        messageElement.innerHTML = `Wspaniała partia! Dowodzone przez Ciebie <b>${winnerName}</b> zmiażdżyły przeciwnika.`;
    } else {
        titleElement.innerText = "💀 KONIEC GRY";
        titleElement.style.color = "#b53434";
        messageElement.innerHTML = `Niestety, Twoje wojska poległy. Zwycięstwo zgarniają <b>${winnerName}</b>.`;
    }

    if (lobbyButton) {
        lobbyButton.onclick = () => { hideGameOverModal(); switchView('lobby'); };
    }
    if (closeButton) {
        closeButton.onclick = () => hideGameOverModal();
    }
    overlay.onclick = (e) => { if (e.target === overlay) hideGameOverModal(); };

    setTimeout(() => { overlay.style.display = 'flex'; }, 400);
}

function handleGameState(payload: any) {
    lastGameStatePayload = payload;
    if (isHistoryMode) return; // nie nadpisuj planszy w trybie historii
    hidePromotionDropdown();

    const state = payload.state;
    const currentUserId = currentPlayer?.id;
    myColor = (payload.white_id === currentUserId) ? 0 : 1;
    currentBudget = state?.Budgets?.[myColor] ?? 0;

    if (state && state.IsOver) {
        console.log("Gra zakończona! Wyłoniono zwycięzcę.");

        // Pobieramy ID lub kolor zwycięzcy
        const winnerColor = state.Winner; // 0 = Biali, 1 = Czarni

        // Odpalamy funkcję końca gry
        handleGameOver(winnerColor);
    }

    console.log("Otrzymano stan gry z Go:", payload);
    if (payload.game_id && payload.game_id !== activeGameId) return;

    const fields = payload.board?.fields;
    if (!fields || !Array.isArray(fields)) return;
    updatePlayerBanner(
        'bottom-player-banner',
        currentUserId!,
        myColor ? 'black' : 'white',
        state.Budgets[myColor],
        'bottom-dice'
    );
    updatePlayerBanner(
        'top-player-banner',
        myColor ? payload.white_id : payload.black_id,
        myColor ? 'white' : 'black',
        state.Budgets[myColor ? 0 : 1],
        'top-dice'
    );
    renderDiceDotsTo('bottom-dice', bottomDiceValue);
    renderDiceDotsTo('top-dice', topDiceValue);
    // Obracamy planszę dla czarnego gracza
    const boardElement = document.getElementById('board');
    if (boardElement) {
        if (myColor === 1) {
            boardElement.classList.add('flipped');
        } else {
            boardElement.classList.remove('flipped');
        }
    }

    // 1. AKTUALIZACJA NAGŁÓWKA (Ruch i Skill)
    const turnInfo = document.getElementById('game-turn-info');
    if (turnInfo && state) {
        const whoMoves = state.ColorToMove === 0 ? "Białe ⬜" : "Czarne ⬛";
        turnInfo.innerHTML = `Ruch: <b>${whoMoves}</b>`;
    }

    // 2. SPRAWDZENIE CZY TO TURA AKTUALNEGO GRACZA
    isMyTurn = state && (
        (state.ColorToMove === 0 && payload.white_id === currentUserId) ||
        (state.ColorToMove === 1 && payload.black_id === currentUserId)
    );

    // 3. ZARZĄDZANIE PRZYCISKAMI (Rzut kostką i Koniec tury)
    const rollButton = document.getElementById('roll-dice-btn') as HTMLButtonElement;
    const endTurnButton = document.getElementById('end-turn-btn') as HTMLButtonElement;

    if (state) {
        // Przycisk rzutu: Moja tura I tura jeszcze NIE rozpoczęta (brak rzutu)
        if (rollButton) {
            console.log(isMyTurn, state.TurnStarted)
            if (isMyTurn && !state.TurnStarted) {
                rollButton.style.visibility = 'visible';
                rollButton.style.display = 'block';
            } else {
                rollButton.style.visibility = 'hidden';
            }
        }

        // Przycisk końca tury: Moja tura I tura JUŻ rozpoczęta (po rzucie)
        if (endTurnButton) {
            if (isMyTurn && state.TurnStarted) {
                endTurnButton.style.visibility = 'visible';
                rollButton.style.display = 'block';
            } else {
                endTurnButton.style.visibility = 'hidden';
            }
        }
    }

    // 4. OBSŁUGA FINiShOWANIA ANIMACJI KOSTKI
    const diceResult = state.LastRoll;
    const activeDiceId = (myColor !== null && state.ColorToMove === myColor) ? 'bottom-dice' : 'top-dice';

    renderDiceDotsTo(activeDiceId, diceResult);
    if (diceAnimationInterval && diceResult > 0) {
        clearInterval(diceAnimationInterval);
        diceAnimationInterval = null;

        const diceElement = document.getElementById(diceAnimElementId);
        if (diceElement) diceElement.classList.remove('shaking');

        if (rollButton) rollButton.disabled = false;
    }

    // 5. RENDEROWANIE PLANSZY I FIGUR
    const squares = document.querySelectorAll('.square');

    for (let row = 0; row < 8; row++) {
        for (let col = 0; col < 8; col++) {
            const index = row * 8 + col;
            const square = squares[index] as HTMLElement;
            if (!square) continue;

            square.innerHTML = '';
            delete square.dataset.promotable;

            const pieceData = fields[row][col];
            if (!pieceData) continue;

            const pieceElement = document.createElement('div');
            const pieceColorClass = pieceData.color === 0 ? 'white' : 'black';
            pieceElement.className = `piece ${pieceColorClass}`;
            pieceElement.innerText = getPieceIcon(pieceData.type);

            square.appendChild(pieceElement);

            if (pieceData.color === myColor) {
                const isPromotable = isMyTurn && state?.TurnStarted &&
                    pieceData.type === 0 &&
                    ((myColor === 0 && row === 0) || (myColor === 1 && row === 7));
                if (isPromotable) {
                    square.dataset.promotable = 'true';
                } else {
                    setupPieceDragAndDrop(pieceElement, index);
                }
            } else {
                pieceElement.style.cursor = 'not-allowed';
            }
        }
    }

}

function updatePlayerBanner(bannerId: string, playerId: number, colorClass: 'white' | 'black', skill: number, diceId: string) {
    const banner = document.getElementById(bannerId);
    if (!banner) return;

    const playerName = getPlayerNameById(playerId);

    banner.innerHTML = `
        <div class="banner-color-cube ${colorClass}"></div>
        <span class="player-name">${playerName}</span>
        <div class="dice mini" id="${diceId}"></div>
        <div class="banner-skill-badge">
            <span class="skill-icon">🎲</span>
            <span class="skill-label">Skill:</span>
            <span class="skill-value">${skill}</span>
        </div>
    `;
}

function getPieceIcon(type: number): string {
    switch (type) {
        case 0: return '♟';
        case 1: return '♝';
        case 2: return '♞';
        case 3: return '♜';
        case 4: return '♛';
        case 5: return '♚';
        default: return '?';
    }
}
function handleLegalMoves(payload: any) {// 1. Najpierw usuwamy wszelkie istniejące kropki, żeby wyczyścić planszę
    clearLegalMoves();

    const moves = payload.moves;
    if (!moves || !Array.isArray(moves)) return;

    // 2. Pobieramy wszystkie kafelki planszy
    const squares = document.querySelectorAll('.square');

    // 3. Dla każdego dozwolonego indeksu dodajemy kropkę
    moves.forEach(coords => {
        const square = squares[coordsToIndex(coords)] as HTMLElement;
        if (square) {
            const hasPiece = square.querySelector('.piece') !== null;
            if (hasPiece) {
                // 3. Jeśli jest tu figura, to w legalnym ruchu oznacza to BICIE -> dajemy pomarańczową klasę
                square.classList.add('capture-target');
            } else {
                // 4. Jeśli pole jest puste -> rysujemy standardową kropkę
                const dot = document.createElement('div');
                dot.className = 'legal-dot';
                square.appendChild(dot);
            }
        }
    });
}

// Pomocnicza funkcja do czyszczenia kropek (będziesz jej używać też przy pointerup)
function clearLegalMoves() {
    document.querySelectorAll('.legal-dot').forEach(dot => dot.remove());
    document.querySelectorAll('.square').forEach(square => {
        square.classList.remove('capture-target');
    });
}

async function fetchActiveGames(): Promise<any[]> {
    if (!authToken || !currentPlayer) return [];

    // Wyciągamy ID aktualnego gracza (obsługujemy małe/wielkie litery na wszelki wypadek)
    const currentUserId = currentPlayer.id;

    try {
        // Doklejamy parametry query dokładnie tak, jak wymaga tego Twój nowy endpoint w Go
        const response = await fetch(`${API_URL}/games?ongoing=true&player_id=${currentUserId}`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': authToken
            }
        });

        if (!response.ok) throw new Error(`Błąd pobierania gier: ${response.status}`);

        const data = await response.json();
        console.log("MOJE TRWAJĄCE GRY Z BACKENDU:", data);

        // Zwracamy tablicę gier (dostosuj, jeśli Go opakowuje to w data.games)
        return data.games || data;
    } catch (error) {
        console.error("Nie udało się pobrać listy moich gier:", error);
        return [];
    }
}

function showPromotionDropdown(squareIndex: number) {
    // Remove any existing dropdown first
    const existing = document.getElementById('promotion-dropdown');
    if (existing) existing.remove();
    document.removeEventListener('click', onPromotionOutsideClick, { capture: true } as EventListenerOptions);

    promotionPawnIndex = squareIndex;

    const squares = document.querySelectorAll('.square');
    const square = squares[squareIndex] as HTMLElement;
    if (!square) return;

    const rect = square.getBoundingClientRect();
    const iconClass = myColor === 0 ? 'white-piece' : 'black-piece';

    const dropdown = document.createElement('div');
    dropdown.id = 'promotion-dropdown';
    dropdown.className = 'promotion-dropdown-popup';

    PROMOTION_PIECES.forEach(p => {
        const btn = document.createElement('button');
        const canAfford = currentBudget >= p.cost;
        btn.className = 'promotion-popup-btn' + (canAfford ? '' : ' unaffordable');
        btn.innerHTML = `<span class="promo-icon ${iconClass}">${p.icon}</span><span class="promo-name">${p.name}</span><span class="promo-cost">${p.cost} pkt</span>`;
        if (canAfford) {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                handlePromotionChoice(p.type);
            });
        }
        dropdown.appendChild(btn);
    });

    document.body.appendChild(dropdown);

    // Position below the square; flip above if too close to bottom
    let top = rect.bottom + 4;
    let left = rect.left;
    if (top + 200 > window.innerHeight) top = rect.top - dropdown.offsetHeight - 4;
    if (left + 180 > window.innerWidth) left = window.innerWidth - 184;
    dropdown.style.top = `${top}px`;
    dropdown.style.left = `${left}px`;

    setTimeout(() => {
        document.addEventListener('click', onPromotionOutsideClick, { capture: true } as EventListenerOptions);
    }, 0);
}

function onPromotionOutsideClick(e: MouseEvent) {
    const dropdown = document.getElementById('promotion-dropdown');
    if (!dropdown || dropdown.contains(e.target as Node)) return;
    document.removeEventListener('click', onPromotionOutsideClick, { capture: true } as EventListenerOptions);
    hidePromotionDropdown();
}

function hidePromotionDropdown() {
    const existing = document.getElementById('promotion-dropdown');
    if (existing) existing.remove();
    document.removeEventListener('click', onPromotionOutsideClick, { capture: true } as EventListenerOptions);
    promotionPawnIndex = null;
}

function handlePromotionChoice(pieceType: number) {
    if (promotionPawnIndex === null || !activeGameId) return;
    const pawnIndex = promotionPawnIndex;
    hidePromotionDropdown();
    sendWSMessage('promote_pawn', {
        game_id: activeGameId,
        at: indexToCoords(pawnIndex),
        promote_to: pieceType
    });
}

// --- HISTORIA RUCHÓW ---

function openHistory() {
    openHistoryPanelOnLoad = true;
    if (!activeGameId || !socket || socket.readyState !== WebSocket.OPEN) return;
    sendWSMessage('get_history', { game_id: activeGameId });
}

function handleHistory(payload: any) {
    historyData = payload.turns || [];
    historyFlatMoves = [];
    historyData.forEach((ht, htIdx) => {
        ht.moves.forEach((m) => {
            const fi = historyFlatMoves.length;
            historyFlatMoves.push({ htIdx, mIdx: fi, label: m.notation, board: m.board });
        });
    });

    if (openHistoryPanelOnLoad) {
        openHistoryPanelOnLoad = false;
        historyCurrentIdx = historyFlatMoves.length > 0 ? historyFlatMoves.length - 1 : -1;
        isHistoryMode = historyFlatMoves.length > 0;
        if (isHistoryMode) renderHistoryBoard(historyFlatMoves[historyCurrentIdx].board);
        showHistoryOverlay();
        updateNavUI();
        return;
    }

    if (pendingNavAction === 'prev') {
        pendingNavAction = null;
        navHistoryPrev();
        return;
    }

    updateNavUI();
}

function navHistoryPrev() {
    if (historyFlatMoves.length === 0) {
        pendingNavAction = 'prev';
        if (activeGameId && socket && socket.readyState === WebSocket.OPEN)
            sendWSMessage('get_history', { game_id: activeGameId });
        return;
    }
    const next = isHistoryMode ? historyCurrentIdx - 1 : historyFlatMoves.length - 1;
    if (next < 0) return;
    historyCurrentIdx = next;
    isHistoryMode = true;
    renderHistoryBoard(historyFlatMoves[historyCurrentIdx].board);
    updateNavUI();
    updateHistoryPanelSelection();
}

function navHistoryNext() {
    if (!isHistoryMode || historyFlatMoves.length === 0) return;
    const next = historyCurrentIdx + 1;
    if (next >= historyFlatMoves.length) {
        navHistoryLatest();
    } else {
        historyCurrentIdx = next;
        renderHistoryBoard(historyFlatMoves[historyCurrentIdx].board);
        updateNavUI();
        updateHistoryPanelSelection();
    }
}

function navHistoryLatest() {
    isHistoryMode = false;
    document.getElementById('history-overlay')?.remove();
    if (lastGameStatePayload) handleGameState(lastGameStatePayload);
    updateNavUI();
}

function updateNavUI() {
    const prevBtn  = document.getElementById('nav-prev')   as HTMLButtonElement | null;
    const nextBtn  = document.getElementById('nav-next')   as HTMLButtonElement | null;
    const latestBtn = document.getElementById('nav-latest') as HTMLButtonElement | null;
    const posEl    = document.getElementById('nav-pos');
    if (!prevBtn || !nextBtn || !latestBtn || !posEl) return;

    const total = historyFlatMoves.length;
    if (isHistoryMode && total > 0) {
        posEl.textContent = `${historyCurrentIdx + 1}/${total}`;
        prevBtn.disabled  = historyCurrentIdx <= 0;
        nextBtn.disabled  = false;
        latestBtn.disabled = false;
    } else {
        posEl.textContent = total > 0 ? `${total}/${total}` : '—';
        prevBtn.disabled  = total === 0;
        nextBtn.disabled  = true;
        latestBtn.disabled = true;
    }
}

function updateHistoryPanelSelection() {
    const list = document.getElementById('history-move-list');
    if (!list) return;
    const total = historyFlatMoves.length;
    list.querySelectorAll<HTMLElement>('.hist-move-btn').forEach((btn, i) => {
        btn.classList.toggle('hist-move-active', i === historyCurrentIdx);
    });
    const prev = document.getElementById('hist-prev') as HTMLButtonElement | null;
    const next = document.getElementById('hist-next') as HTMLButtonElement | null;
    const pos  = document.querySelector<HTMLElement>('.hist-pos');
    if (prev) prev.disabled = historyCurrentIdx <= 0;
    if (next) next.disabled = historyCurrentIdx >= total - 1;
    if (pos)  pos.textContent = `${historyCurrentIdx + 1} / ${total}`;
    setTimeout(() => document.querySelector('.hist-move-active')?.scrollIntoView({ block: 'nearest' }), 0);
}

function showHistoryOverlay() {
    document.getElementById('history-overlay')?.remove();

    const overlay = document.createElement('div');
    overlay.id = 'history-overlay';
    overlay.className = 'history-overlay';

    const total = historyFlatMoves.length;
    const cur = historyCurrentIdx;

    let listHTML = '';
    let flatIdx = 0;
    historyData.forEach((ht) => {
        const colorLabel = ht.color === 0 ? '⬜' : '⬛';
        listHTML += `<div class="hist-halfturn-header">${ht.num}. ${colorLabel} [🎲${ht.roll}]</div>`;
        ht.moves.forEach((m) => {
            const fi = flatIdx;
            const isActive = fi === cur ? ' hist-move-active' : '';
            listHTML += `<button class="hist-move-btn${isActive}" data-fidx="${fi}">${m.notation}</button>`;
            flatIdx++;
        });
    });

    overlay.innerHTML = `
        <div class="history-panel">
            <div class="history-header">
                <span class="history-title">Ruchy</span>
                <button class="modal-close-btn" id="history-close-btn">✕</button>
            </div>
            <div class="history-move-list" id="history-move-list">${listHTML || '<span style="color:#797977;font-size:13px;">Brak ruchów</span>'}</div>
            <div class="history-nav">
                <button id="hist-prev" class="hist-nav-btn" ${cur <= 0 ? 'disabled' : ''}>◀</button>
                <span class="hist-pos">${total > 0 ? cur + 1 : 0} / ${total}</span>
                <button id="hist-next" class="hist-nav-btn" ${cur >= total - 1 ? 'disabled' : ''}>▶</button>
                <button id="hist-exit-btn" class="rules-button" style="margin-left:8px;">Zamknij</button>
            </div>
        </div>
    `;

    document.body.appendChild(overlay);

    document.getElementById('history-close-btn')!.addEventListener('click', navHistoryLatest);
    document.getElementById('hist-exit-btn')!.addEventListener('click', navHistoryLatest);

    document.getElementById('hist-prev')?.addEventListener('click', () => {
        navHistoryPrev();
    });
    document.getElementById('hist-next')?.addEventListener('click', () => {
        navHistoryNext();
    });

    document.getElementById('history-move-list')?.querySelectorAll('.hist-move-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const fi = parseInt((btn as HTMLElement).dataset.fidx!);
            historyCurrentIdx = fi;
            isHistoryMode = true;
            renderHistoryBoard(historyFlatMoves[fi].board);
            updateNavUI();
            updateHistoryPanelSelection();
        });
    });

    setTimeout(() => document.querySelector('.hist-move-active')?.scrollIntoView({ block: 'nearest' }), 0);
}

function renderHistoryBoard(boardJSON: string) {
    try {
        const boardData = JSON.parse(boardJSON);
        const fields: any[][] = boardData.fields;
        const squares = document.querySelectorAll('.square');
        for (let row = 0; row < 8; row++) {
            for (let col = 0; col < 8; col++) {
                const index = row * 8 + col;
                const square = squares[index] as HTMLElement;
                if (!square) continue;
                square.innerHTML = '';
                const pieceData = fields[row][col];
                if (!pieceData) continue;
                const pieceElement = document.createElement('div');
                pieceElement.className = `piece ${pieceData.color === 0 ? 'white' : 'black'}`;
                pieceElement.innerText = getPieceIcon(pieceData.type);
                square.appendChild(pieceElement);
            }
        }
    } catch (e) {
        console.error('Błąd renderowania historycznej planszy:', e);
    }
}

(window as any).openHistory = openHistory;
(window as any).navHistoryPrev = navHistoryPrev;
(window as any).navHistoryNext = navHistoryNext;
(window as any).navHistoryLatest = navHistoryLatest;

// --- SYSTEM PRZEŁĄCZANIA WIDOKÓW (ROUTER) ---
export function switchView(viewName: 'login' | 'lobby' | 'game') {
    if (viewName === 'login') initLogin();
    if (viewName === 'lobby') initLobby(); // Wywołanie async zainicjuje pobieranie
    if (viewName === 'game') initGame();
}

// Rejestrujemy funkcję globalnie, żeby działała w atrybutach onclick w HTML
(window as any).switchView = switchView;

// Na starcie sprawdzamy, czy gracz ma już token sesji
if (authToken) {
    console.log("Znaleziono istniejący token, przekierowuję do lobby...");
    initWebSocket(); // <-- ODPALAMY WS TUTAJ
    switchView('lobby');
} else {
    switchView('login');
}