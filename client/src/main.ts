const app = document.getElementById('app')!;

// --- 1. MOCKUP WIDOKU LOGOWANIA ---
const loginHTML = `
    <div class="screen">
        <h2>Szachy z Kostką 🎲</h2>
        <input type="text" id="username" placeholder="Wpisz swój nick..." />
        <button id="login-btn">Wejdź do gry</button>
    </div>
`;

function initLogin() {
    app.innerHTML = loginHTML;
    document.getElementById('login-btn')?.addEventListener('click', () => {
        const name = (document.getElementById('username') as HTMLInputElement).value;
        if (name) {
            alert(`Zalogowano jako: ${name} (Zapisano do makiety)`);
            switchView('lobby'); // Przejdź do lobby po "zalogowaniu"
        }
    });
}

// --- 2. MOCKUP WIDOKU LOBBY (ZAKTUALIZOWANY) ---

// Lista sztucznych graczy dostępnych online do testów
const mockUsers = ["Arek", "Kamil", "Monika", "Patryk_99", "Zosia"];

function initLobby() {
    // Generujemy HTML dla listy graczy z tablicy
    const usersListHTML = mockUsers
        .map(user => `
            <li class="player-item" data-username="${user}">
                🟢 ${user} <button class="play-with-btn">Graj</button>
            </li>
        `).join('');

    app.innerHTML = `
        <div class="screen">
            <h3>Witaj w Lobby!</h3>
            
            <!-- Sekcja 1: Wpisanie nicku ręcznie -->
            <div class="lobby-section">
                <label for="search-player">Wpisz nick gracza:</label>
                <div style="display: flex; gap: 10px; margin-top: 5px;">
                    <input type="text" id="search-player" placeholder="Np. Janek..." style="flex: 1;" />
                    <button id="start-game-manual-btn">Zagraj</button>
                </div>
            </div>

            <hr style="width: 100%; border: 0; border-top: 1px solid #555; margin: 15px 0;" />

            <!-- Sekcja 2: Wybór z listy graczy online -->
            <div class="lobby-section">
                <h4>Gracze online (${mockUsers.length}):</h4>
                <ul id="online-players-list">
                    ${usersListHTML}
                </ul>
            </div>

            <hr style="width: 100%; border: 0; border-top: 1px solid #555; margin: 15px 0;" />

            <!-- Sekcja 3: Twoje aktywne gry -->
            <div class="lobby-section">
                <h4>Twoje aktywne gry:</h4>
                <ul>
                    <li>Michał vs Ty <button onclick="window.switchView('game')">Graj (Twój ruch)</button></li>
                </ul>
            </div>
        </div>
    `;

    // --- LOGIKA OBSŁUGI ZDARZEŃ ---

    // 1. Obsługa wpisania ręcznego
    const manualBtn = document.getElementById('start-game-manual-btn')!;
    const searchInput = document.getElementById('search-player') as HTMLInputElement;

    manualBtn.addEventListener('click', () => {
        const opponentName = searchInput.value.trim();
        if (opponentName) {
            alert(`Rozpoczynasz grę z graczem: ${opponentName} (Wpisany ręcznie)`);
            switchView('game');
        } else {
            alert('Wpisz najpierw nick gracza!');
        }
    });

    // 2. Obsługa kliknięcia w gracza z listy (Delegacja zdarzeń)
    const playersList = document.getElementById('online-players-list')!;
    playersList.addEventListener('click', (event) => {
        const target = event.target as HTMLElement;

        // Sprawdzamy czy kliknięto przycisk "Graj" lub sam element listy
        const listItem = target.closest('.player-item') as HTMLElement;

        if (listItem) {
            const opponentName = listItem.dataset.username;
            alert(`Rozpoczynasz grę z graczem: ${opponentName} (Wybrany z listy)`);
            switchView('game');
        }
    });
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

    // 4. Wspólna funkcja wykonująca fizyczne przeniesienie figury w DOM
    function executeMove(fromIndex: number, toIndex: number) {
        const fromSquare = board.querySelector(`[data-index="${fromIndex}"]`) as HTMLElement;
        const toSquare = board.querySelector(`[data-index="${toIndex}"]`) as HTMLElement;
        const movingPiece = fromSquare.querySelector('.piece');

        if (movingPiece) {
            // Jeśli na docelowym polu stoi figura, bijemy ją (usuwamy z makiety)
            const targetPiece = toSquare.querySelector('.piece');
            if (targetPiece) {
                toSquare.removeChild(targetPiece);
            }

            // Przenosimy element figury do nowego kafelka
            toSquare.appendChild(movingPiece);

            // Aktualizujemy indeks startowy w funkcji drag&drop dla tej figury
            // Wyłączamy całkowicie natywny mechanizm przeciągania HTML5 dla tej figury
            movingPiece.addEventListener('dragstart', (e) => e.preventDefault());
            setupPieceDragAndDrop(movingPiece as HTMLElement, toIndex);
        }
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

// --- SYSTEM PRZEŁĄCZANIA WIDOKÓW (ROUTER) ---
export function switchView(viewName: 'login' | 'lobby' | 'game') {
    if (viewName === 'login') initLogin();
    if (viewName === 'lobby') initLobby();
    if (viewName === 'game') initGame();
}

// Rejestrujemy funkcję globalnie, żeby działała w atrybutach onclick w HTML
(window as any).switchView = switchView;

// Uruchomienie na starcie widoku logowania
switchView('login');