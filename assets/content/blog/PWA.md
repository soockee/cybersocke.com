---
name: "Progressive Web Apps"
slug: "progressive-web-apps"
tags: ["web"]
date: 2021-02-19
description: "An introduction to Progressive Web Apps (PWAs) and their features."
---

The concept of Progressive Web Apps (PWAs) stems from **progressive enhancement**. The more functionality a browser can support, the better the experience delivered to the user.

## Definition

```
Progressive Web Apps are responsive web applications transmitted via HTTPS. Following the principles of progressive enhancement, they leverage browser capabilities for incremental improvements. Features like offline functionality through Service Workers, installation via a Web App Manifest, and push notifications ensure a reliable, engaging, and native-like user experience.
```

## Features

A Progressive Web App is a flexible, adaptive application built entirely with web technologies. It should possess the following characteristics:

- **Discoverable**
- **Installable**
- **Linkable**
- **Network Independent**
- **Progressive**
- **Re-engageable**
- **Responsive**
- **Safe**

PWAs are installable directly from the browser. Developers can publish updates seamlessly, with users receiving notifications about available updates via push notifications. Browser compatibility is generally excellent.

## Popular Frameworks

- React
- Vue.js
- Angular
- Preact
- Ember
- Svelte

## Tools

- **PWA Builder**
- **Lighthouse**

# Introduction to Vue.js

Vue.js is a client-side JavaScript framework for building user interfaces and single-page applications. Inspired by the MVVM pattern, it is available in versions 2.6.11 and 3.0.5.



<img src="https://cybersocke.com/assets/blog/img/pwa/MVVMPattern.png" 
alt="mvvp pattern" style="max-width: 100%; height: auto;">


## Hello Vue Example

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Hello World - Vue.js</title>
    <script type="text/javascript" src="https://unpkg.com/vue@next"></script>
</head>
<body>
    <div id="app">{{ message }}</div>
    <script>
        const HelloVueApp = {
            data() {
                return {
                    message: 'Hello World'
                }
            }
        }
        Vue.createApp(HelloVueApp).mount('#app')
    </script>
</body>
</html>
```

## Application Structure

Every Vue application has a root component. Additional components are structured within this root.

<img src="https://cybersocke.com/assets/blog/img/pwa/struct.png" 
alt="Struct of root component" style="max-width: 100%; height: auto;">


Components can be registered globally or locally:

```javascript
// Global registration
app.component('todo-item', {
    template: `<li>This is a to-do item</li>`
})

// Local registration
const ComponentA = {
    /* ... */
}

const ComponentB = {
    components: {
        'component-a': ComponentA
    }
}
```

Usage:

```html
<ol>
    <todo-item></todo-item>
</ol>
```

## Component Example

```javascript
app.component('button-counter', {
    data() {
        return {
            count: 0
        }
    },
    template: `
        <button @click="count++">
            You clicked me {{ count }} times.
        </button>
    `
})
```

<img src="https://cybersocke.com/assets/blog/img/pwa/counter.png" 
alt="counter ui element" style="max-width: 100%; height: auto;">


## Methods

```javascript
const app = Vue.createApp({
    data() {
        return {
            count: 0
        }
    },
    methods: {
        increment() {
            this.count++
        }
    }
})

const vm = app.mount('#app')

console.log(vm.count) // => 4
vm.increment()
console.log(vm.count) // => 5
```

In a view:

```html
<button @click="increment">Up Vote</button>
```

## Lifecycle Hooks

Vue includes a lifecycle system for components, similar to frameworks like React or Android development.

<img src="https://cybersocke.com/assets/blog/img/pwa/lifecycle.png" 
alt="lifecycle" style="max-width: 100%; height: auto;">


## Props

Vue supports props for passing data to components, similar to React:

```javascript
app.component('blog-post', {
    props: ['title'],
    template: `<h4>{{ title }}</h4>`
})
```

## List Rendering

```javascript
Vue.createApp({
    data() {
        return {
            myObject: {
                title: 'Example Title',
                author: 'Example Author',
                publishDate: '2021'
            }
        }
    }
}).mount('#v-for-object')
```

```html
<ul id="v-for-object" class="demo">
    <li v-for="value in myObject">
        {{ value }}
    </li>
</ul>
```

## Conditional Rendering

```html
<div v-if="Math.random() > 0.5">
    Now you see me
</div>
<div v-else>
    Now you don't
</div>
```

## v-model

The `v-model` directive enables two-way data binding, as seen in Vue 2:

```html
<ChildComponent v-model="pageTitle" />
```

### Source

This content is based on a seminar conducted as part of the Web Technology module at THM.

Presenters:
- John Deutesfeld
- Marlin Brandstädter
- Nils Mittler
- Stefanie Josefine Antonia Lehn

