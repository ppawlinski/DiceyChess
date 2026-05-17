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

// --- 1. WIDOK LOGOWANIA ---

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

        alert(`Gra o ID ${gameId} została utworzona! Przełączam na widok szachownicy.`);

        // Przełączamy użytkownika do ekranu gry
        switchView('game');

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
                    handleLegalMoves(message.payload);
                    break;
                case 'error':
                    alert(`Błąd serwera (WS): ${message.payload.error}`);
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
            <h3>Witaj w Lobby, <span id="lobby-username" style="color: #4caf50;">Gracz</span>!</h3>
            
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
                <button class="play-with-btn">Wyzwij</button>
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
            </li>
        `).join('');
    }

    // 3. Renderowanie trwających gier z zamianą ID na Nicknamy
    if (activeGames.length === 0) {
        gamesList.innerHTML = `<li style="color: #aaa; font-style: italic; font-size: 14px;">Nie uczestniczysz w żadnej grze.</li>`;
    } else {
        gamesList.innerHTML = activeGames.map(g => {
            const gameId = g.game_id || g.id || g.ID;

            // Wyciągamy ID białego i czarnego gracza z struktury Go
            const whiteId = g.white_id || g.WhiteID || g.white_player_id;
            const blackId = g.black_id || g.BlackID || g.black_player_id;

            // Mapujemy ID na czytelne nicki
            const whiteName = getPlayerNameById(whiteId);
            const blackName = getPlayerNameById(blackId);

            return `
                <li style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; background: #2a2a2a; padding: 6px 10px; border-radius: 4px;">
                    <span>🏆 Mecz #${gameId}: <b style="color: #fff;">${whiteName}</b> vs <b>${blackName}</b></span>
                    <button class="join-game-btn" data-game-id="${gameId}">Dołącz</button>
                </li>
            `;
        }).join('');
    }

    setupLobbyListEvents();
}

// Zmiana w podpinaniu eventu listy (uproszczenie, bo nasłuchujemy na online-players-list)
function setupLobbyListEvents() {
    document.getElementById('online-players-list')?.addEventListener('click', async (e) => {
        const target = e.target as HTMLElement;
        if (target.classList.contains('play-with-btn')) {
            const item = target.closest('.player-item') as HTMLElement;
            const oppId = parseInt(item?.dataset.id || '');
            if (oppId) await createGame(oppId);
        }
    });

    document.getElementById('active-games-list')?.addEventListener('click', (e) => {
        const target = e.target as HTMLElement;
        if (target.classList.contains('join-game-btn')) {
            const gameId = parseInt(target.dataset.gameId || '');
            if (gameId) joinGame(gameId);
        }
    });
}

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
            <div class="dice-section">
                <span>Ruch gracza: <b>Ty</b></span>
                <button id="roll-btn">🎲 Rzuć kostką</button>
                <div class="dice" id="dice-element"></div>
            </div>
            <div class="chessboard" id="board"></div>
            <div><button onclick="window.switchView('lobby')">🏳️ Poddaj się</button></div>
        </div>
    `;

    const board = document.getElementById('board')!;

    // Stan gry na potrzeby makiety (wybrana pozycja startowa)
    let selectedSquareIndex: number | null = null;

    const startingPieces: { [key: number]: string } = {
        0: '♜', 1: '♞', 2: '♝', 3: '♛', 4: '♚', 5: '♝', 6: '♞', 7: '♜',
        8: '♟', 9: '♟', 10: '♟', 11: '♟', 12: '♟', 13: '♟', 14: '♟', 15: '♟',
        48: '♙', 49: '♙', 50: '♙', 51: '♙', 52: '♙', 53: '♙', 54: '♙', 55: '♙',
        56: '♖', 57: '♘', 58: '♗', 59: '♕', 60: '♔', 61: '♗', 62: '♘', 63: '♖'
    };

    // 1. Generowanie planszy 8x8
    for (let i = 0; i < 64; i++) {
        const row = Math.floor(i / 8);
        const col = i % 8;
        const isLight = (row + col) % 2 === 0;

        const square = document.createElement('div');
        square.classList.add('square', isLight ? 'light' : 'dark');
        square.dataset.index = i.toString();

        if (startingPieces[i]) {
            // Każdą figurę zawijamy w osobny span, aby łatwo ją przesuwać (Drag&Drop)
            const piece = document.createElement('span');
            piece.classList.add('piece');
            piece.textContent = startingPieces[i];
            square.appendChild(piece);

            // Podpinamy zdarzenia Pointer Events do figury
            setupPieceDragAndDrop(piece, i);
        }

        // Mechanizm obsługi kliknięć (alternatywa dla Drag&Drop, idealna na Mobile)
        square.addEventListener('click', (e) => {
            // Jeśli kliknięto w figurę, ignorujemy ten handler, bo obsłuży go Drag&Drop
            if ((e.target as HTMLElement).classList.contains('piece') && selectedSquareIndex === null) return;

            handleSquareClick(i);
        });

        board.appendChild(square);
    }

    // 2. Obsługa logiki klikania (Wybór -> Ruch)
    function handleSquareClick(clickedIndex: number) {
        const clickedSquare = board.querySelector(`[data-index="${clickedIndex}"]`) as HTMLElement;
        const currentSelected = board.querySelector('.selected') as HTMLElement;

        if (selectedSquareIndex === null) {
            // KROK 1: Wybór figury poprzez kliknięcie
            if (clickedSquare.textContent.trim() !== "") {
                selectedSquareIndex = clickedIndex;
                clickedSquare.classList.add('selected');
            }
        } else {
            // KROK 2: Wybór pola docelowego
            if (selectedSquareIndex === clickedIndex) {
                // Kliknięcie w to samo pole - odznaczamy
                clickedSquare.classList.remove('selected');
                selectedSquareIndex = null;
            } else {
                // Wykonanie ruchu w makiecie
                executeMove(selectedSquareIndex, clickedIndex);
                if (currentSelected) currentSelected.classList.remove('selected');
                selectedSquareIndex = null;
            }
        }
    }

    // 3. Obsługa Drag & Drop za pomocą Pointer Events (PC & Mobile)
    function setupPieceDragAndDrop(piece: HTMLElement, startIndex: number) {
        piece.addEventListener('dragstart', (e) => e.preventDefault());

        // Nasłuchujemy na parentSquare (czyli na kafelku), bo figura ma pointer-events: none
        const parentSquare = piece.parentElement!;

        parentSquare.addEventListener('pointerdown', (e) => {
            // Interweniujemy tylko, jeśli na kafelku jest nasza figura
            if (!parentSquare.contains(piece)) return;

            const initialIndex = parseInt(parentSquare.dataset.index!);
            const startX = e.clientX;
            const startY = e.clientY;

            let isDragging = false;
            const rect = piece.getBoundingClientRect();

            const moveAt = (clientX: number, clientY: number) => {
                piece.style.left = clientX - rect.width / 2 + 'px';
                piece.style.top = clientY - rect.height / 2 + 'px';
            };

            function onPointerMove(ev: PointerEvent) {
                const deltaX = ev.clientX - startX;
                const deltaY = ev.clientY - startY;
                const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);

                // Jeśli ruch jest mniejszy niż 7 pikseli, pozwalamy działać zwykłemu kliknięciu na kafelku
                if (!isDragging && distance < 7) {
                    return;
                }

                if (!isDragging) {
                    isDragging = true;
                    e.stopPropagation(); // Blokujemy aktywację kliknięcia, bo zaczynamy drag

                    document.querySelectorAll('.square').forEach(s => s.classList.remove('selected'));
                    parentSquare.classList.add('selected');

                    // Aktywujemy fizyczność figury na czas przeciągania
                    piece.classList.add('dragging');
                    piece.style.width = rect.width + 'px';
                    piece.style.height = rect.height + 'px';
                    piece.style.position = 'fixed';
                    piece.style.zIndex = '1000';
                }

                moveAt(ev.clientX, ev.clientY);
            }

            document.addEventListener('pointermove', onPointerMove);

            const onPointerUp = function (ev: PointerEvent) {
                document.removeEventListener('pointermove', onPointerMove);
                document.removeEventListener('pointerup', onPointerUp);

                if (!isDragging) {
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

                if (targetSquare) {
                    const targetIndex = parseInt(targetSquare.dataset.index!);
                    if (initialIndex !== targetIndex) {
                        executeMove(initialIndex, targetIndex);
                    }
                }
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
    // 4. Wspólna funkcja wykonująca fizyczne przeniesienie figury w DOM
    function executeMove(fromIndex: number, toIndex: number) {

        if (!activeGameId) {
            console.error("Brak aktywnego ID gry!");
            return;
        }
        const fromSquare = board.querySelector(`[data-index="${fromIndex}"]`) as HTMLElement;
        const toSquare = board.querySelector(`[data-index="${toIndex}"]`) as HTMLElement;
        const movingPiece = fromSquare.querySelector('.piece');

        // Wysyłamy ruch do serwera w formacie:
        // {"type": "make_move", "payload": {"game_id": 1, "from": {"Row": 6, "Col": 0}, "to": {"Row": 5, "Col": 0}}}
        sendWSMessage("make_move", {
            game_id: activeGameId,
            from: indexToCoords(fromIndex),
            to: indexToCoords(toIndex)
        });
    }

    // --- LOGIKA REPRODUKCJI KROPEK NA KOSTCE (BEZ ZMIAN) ---
    const diceElement = document.getElementById('dice-element')!;
    const rollBtn = document.getElementById('roll-btn') as HTMLButtonElement;

    const dotPositions: { [key: number]: number[] } = {
        1: [4], 2: [0, 8], 3: [0, 4, 8], 4: [0, 2, 6, 8], 5: [0, 2, 4, 6, 8], 6: [0, 2, 3, 5, 6, 8]
    };

    function renderDiceDots(value: number) {
        diceElement.innerHTML = '';
        for (let i = 0; i < 9; i++) {
            const cell = document.createElement('div');
            if (dotPositions[value].includes(i)) {
                const dot = document.createElement('div');
                dot.classList.add('dot');
                cell.appendChild(dot);
            }
            diceElement.appendChild(cell);
        }
    }

    renderDiceDots(1);

    rollBtn.addEventListener('click', () => {
        rollBtn.disabled = true;
        diceElement.classList.add('shaking');
        let counter = 0;

        const interval = setInterval(() => {
            renderDiceDots(Math.floor(Math.random() * 6) + 1);
            counter++;
            if (counter > 12) {
                clearInterval(interval);
                diceElement.classList.remove('shaking');
                const finalValue = Math.floor(Math.random() * 6) + 1;
                renderDiceDots(finalValue);
                rollBtn.disabled = false;
                const figureNames = ["Pionek", "Skoczek", "Goniec", "Wieża", "Hetman", "Król"];
                alert(`Wylosowano: ${finalValue} oczek. W tym ruchu możesz ruszyć się tylko: ${figureNames[finalValue - 1]}iem!`);
            }
        }, 70);
    });
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
    console.log("Gra wystartowała! Szczegóły:", payload);
    // Zapisujemy ID gry z payloadu
    activeGameId = payload.game_id || payload.id;
    switchView('game');
}

function handleGameState(payload: any) {
    console.log("Nowy stan planszy z serwera:", payload);
    // Tutaj wepniemy funkcję, która przerysuje naszą szachownicę na podstawie tablicy z Go
}

function handleLegalMoves(payload: any) {
    console.log("Możliwe ruchy dla wybranej figury:", payload);
    // Tutaj podświetlimy kropkami kafelki, na które figura może skoczyć
}

async function fetchActiveGames(): Promise<any[]> {
    if (!authToken) return [];
    try {
        const response = await fetch(`${API_URL}/games`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                // Jeśli serwer wymaga autoryzacji do pobrania listy gier:
                'Authorization': authToken
            }
        });

        if (!response.ok) throw new Error(`Błąd pobierania gier: ${response.status}`);

        const data = await response.json();
        console.log("AKTYWNE GRY Z REST:", data);

        // Zwracamy tablicę (dostosuj jeśli serwer opakowuje to w data.games)
        return data.games || data;
    } catch (error) {
        console.error("Nie udało się pobrać listy gier:", error);
        return [];
    }
}

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