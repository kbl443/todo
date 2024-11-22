import {Events} from "@wailsio/runtime";


window.addEventListener("load", function() {
    handleHashChange();
});

const defaultView = document.getElementById('default-view');
const dynamicView = document.getElementById('dynamic-view');



//////////////////////////////////////////
// if no hash it will show default view, request all items
// if hash it will show dynamic view, request item with id = hash
//////////////////////////////////////////
function handleHashChange() {
  const hash = window.location.hash.slice(1);

  if (hash) {
    // Show the dynamic view
    dynamicView.style.display = 'block';
    defaultView.style.display = 'none';

    // Check if the requested item already has a corresponding element
    const existingItemElement = document.querySelector(`[data-item-id="${hash}"]`);
    if (existingItemElement) {
      // The item is already displayed, no need to request it again
      return;
    }

    // Hash contains an ID, emit an event to the backend
    Events.Emit({ name: 'requestSingleItem', data: hash });
  } else {
    // No hash, show the default view
    dynamicView.style.display = 'none';
    defaultView.style.display = 'block';

    Events.Emit({ name: 'requestTodos', data: 'useless' });
  }
}

//////////////////////////////////////////
// Submit form to create a new item
//////////////////////////////////////////
document.getElementById('todo-form').addEventListener('submit', function(event) {
  event.preventDefault();

  var title = document.getElementById('title').value;
  var description = document.getElementById('description').value;
  var dueDate = document.getElementById('dueDate').value;

  var todoItem = {
      title: title,
      description: description,
      completed: false,
      dueDate: dueDate
  };
  // stringify the object so it's sent as JSON string
  Events.Emit({ name: 'createTodo', data: JSON.stringify(todoItem) });


  // Clear the form
  document.getElementById('title').value = '';
  document.getElementById('description').value = '';
  document.getElementById('dueDate').value = '';
});


//////////////////////////////////////////
// listener to show a list of all items
//////////////////////////////////////////
Events.On('responseTodos', (event) => {
  let todoItems;
  try {
    todoItems = JSON.parse(event.data[0]);
  } catch (error) {
    // If the JSON parsing fails, it means there's a message instead of an array
    const noItemsMessage = event.data[0];
    const todoListElement = document.getElementById('todo-list');
    todoListElement.innerHTML = '';

    const messageElement = document.createElement('p');
    messageElement.textContent = noItemsMessage;
    todoListElement.appendChild(messageElement);
    return;
  }

  const todoListElement = document.getElementById('todo-list');
  todoListElement.innerHTML = '';

  if (!todoItems || todoItems.length === 0) {
    // If there are no todo items, display the "create your first element" message
    const noItemsMessage = document.createElement('p');
    noItemsMessage.textContent = 'nothing todo here';
    todoListElement.appendChild(noItemsMessage);
  } else {
    // Sort the todoItems array based on a specific property
    todoItems.sort((a, b) => {
      // Sort by due date in ascending order
      return new Date(a.dueDate) - new Date(b.dueDate);
    });

    // Render the todo items
    todoItems.forEach(item => {
      const template = document.getElementById('todo-item-template');
      const clone = template.content.cloneNode(true);
      const listItem = clone.querySelector('.todo-item');
      listItem.dataset.itemId = item.id;
      const titleElement = clone.querySelector('.todo-title');
      const dueDateElement = clone.querySelector('.todo-due-date');

      if (item.completed) {
        listItem.classList.add('completed-item');
        titleElement.classList.add('completed-title');
        dueDateElement.classList.add('completed-date');
      } else {
        listItem.classList.add('pending-item');
      }

      titleElement.textContent = item.title;
      var formattedDueDate = formatDueDate(item.dueDate);
      dueDateElement.textContent = `${formattedDueDate}`;

      const detailsElement = clone.querySelector('.details');
      detailsElement.addEventListener('click', () => {
        Events.Emit({ name: 'openItemWindow', data: item.id });
      });

      const deleteElement = clone.querySelector('.delete');
      deleteElement.addEventListener('click', () => {
        Events.Emit({ name: 'deleteTodo', data: item.id });
      });

      todoListElement.appendChild(clone);
    });


  }
});



//////////////////////////////////////////
// listener to show a single requested item
//////////////////////////////////////////
Events.On('responseSingleItem', (event) => {
  const {data} = event;
  console.log(data);
  if (data !== null) {
    const todoItem = JSON.parse(data[0]);
    if (todoItem !== null) {

      const singleTodoElement = document.createElement('div');
      singleTodoElement.classList.add('single-todo');
      singleTodoElement.dataset.itemId = todoItem.id;

      singleTodoElement.innerHTML = `
      <div class="todo-actions">
        <button class="todo-description-edit">Save</button>
        <button class="cancel-edit">close</button>
      </div>
      <div class="todo-header">
        <div class="header-left">
          <h3 class="todo-title">${todoItem.title}</h3>
          <p class="todo-due-date">Due: ${todoItem.dueDate}</p>
        </div>
        <div class="header-right">
          <input type="checkbox" class="todo-completed" ${todoItem.completed ? 'checked' : ''}>
        </div>
      </div>
      <div class="todo-content">
        <div class="todo-description" contenteditable="plaintext-only">${todoItem.description}</div>
      </div>
    `;
    

      // Checkbox toggle
      singleTodoElement.querySelector('.todo-completed').addEventListener('change', () => {
        Events.Emit({ name: 'toggleTodoCompleted', data: todoItem.id });
      });
      
        // Description edit
      singleTodoElement.querySelector('.todo-description-edit').addEventListener('click', () => {
        const editData = [
          todoItem.id, 
          singleTodoElement.querySelector('.todo-description').innerHTML
        ];
        Events.Emit({ name: 'editDescription', data: editData });
      });
      
      // Close window
      singleTodoElement.querySelector('.cancel-edit').addEventListener('click', () => {
        Events.Emit({ name: 'close-window', data: 'useless' });
      });
      
      //////////////////////////////////////////
      // Showing the element "Updated!!!"
      //////////////////////////////////////////
      Events.On('feedbackSaved', (event) => {
        // Check if the current window's item ID matches the received item ID
        if (Number(window.location.hash.slice(1)) === event.data[0]) {
          console.log('showing feedback "updated!"');
      
          // Create the feedback element
          const feedbackElement = document.createElement('div');
          feedbackElement.classList.add('feedback-note');
          feedbackElement.textContent = 'Updated!!!';
      
          // Add the feedback element to the DOM
          singleTodoElement.appendChild(feedbackElement);
      
          // Show the feedback element
          setTimeout(() => {
            feedbackElement.style.opacity = '1';
          }, 100);
      
          // Hide the feedback element after 1 second
          setTimeout(() => {
            feedbackElement.style.opacity = '0';
            setTimeout(() => {
              singleTodoElement.removeChild(feedbackElement);
            }, 500);
          }, 1000);
        }
      });
      

      if (!window.isFirstItemShown) {
        dynamicView.appendChild(singleTodoElement);
        window.isFirstItemShown = true;
      }
    } else {
      console.log(1);
      const singleTodoElement = document.createElement('p');
      singleTodoElement.classList.add('error');
      singleTodoElement.innerText = 'did you just delete the item???';
      // to close the current window
      const cancelElement = document.createElement('button');
      cancelElement.classList.add('cancel-edit');
      cancelElement.textContent = 'close';
      cancelElement.addEventListener('click', () => {
        
        Events.Emit({ name: 'close-window', data: 'useless' });
      });
      dynamicView.appendChild(singleTodoElement);
      dynamicView.appendChild(cancelElement);
    }
  } else {
    console.log(2);
    const singleTodoElement = document.createElement('p');
    singleTodoElement.classList.add('error');
    singleTodoElement.innerText = 'The requested item could not be found. Please try again.';
    // to close the current window
    const cancelElement = document.createElement('button');
    cancelElement.classList.add('cancel-edit');
    cancelElement.textContent = 'close';
    cancelElement.addEventListener('click', () => {
      
      Events.Emit({ name: 'close-window', data: 'useless' });
    });
    dynamicView.appendChild(singleTodoElement);
    dynamicView.appendChild(cancelElement);
  }
});



//////////////////////////////////////////
// Showing the element "Created!!!"
//////////////////////////////////////////
Events.On('feedbackCreated', (event) => {
  // Create the feedback element
  const feedbackElement = document.createElement('div');
  feedbackElement.classList.add('feedback-note');
  feedbackElement.textContent = event.data;

  // Add the feedback element to the DOM
  defaultView.appendChild(feedbackElement);

  // Show the feedback element
  setTimeout(() => {
    feedbackElement.style.opacity = '1';
  }, 100);

  // Hide the feedback element after 1 second
  setTimeout(() => {
    feedbackElement.style.opacity = '0';
    setTimeout(() => {
      defaultView.removeChild(feedbackElement);
    }, 500);
  }, 1000);
});




//////////////////////////////////////////
// Theme toggle
//////////////////////////////////////////
const themeToggleBtn = document.getElementById('theme-toggle-btn');
const themeIcon = themeToggleBtn.querySelector('.theme-icon');

// Check for saved theme preference
const savedTheme = localStorage.getItem('theme') || 'dark';
document.documentElement.setAttribute('data-theme', savedTheme);
updateThemeIcon(savedTheme);

themeToggleBtn.addEventListener('click', () => {
    const currentTheme = document.documentElement.getAttribute('data-theme');
    const newTheme = currentTheme === 'light' ? 'dark' : 'light';
    
    document.documentElement.setAttribute('data-theme', newTheme);
    localStorage.setItem('theme', newTheme);
    updateThemeIcon(newTheme);
});

function updateThemeIcon(theme) {
    themeIcon.textContent = theme === 'light' ? '🌙' : '☀️';
}


//////////////////////////////////////////
// due date format
//////////////////////////////////////////
function formatDueDate(dueDate) {
  var now = new Date();
  var due = new Date(dueDate);
  var diffInDays = Math.ceil((due - now) / (1000 * 60 * 60 * 24));

  if (diffInDays < 0) {
    return 'Overdue';
  } else if (diffInDays === 0) {
    return 'Due today';
  } else if (diffInDays === 1) {
    return 'Due tomorrow';
  } else {
    return `Due in ${diffInDays} days`;
  }
}



//////////////////////////////////////////
// time
//////////////////////////////////////////
const timeElement = document.getElementById('time');
Events.On('time', (time) => {
    timeElement.innerText = time.data;
});
